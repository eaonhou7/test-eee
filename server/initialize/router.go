package initialize

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/flipped-aurora/gin-vue-admin/server/docs"
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/middleware"
	"github.com/flipped-aurora/gin-vue-admin/server/router"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

type justFilesFilesystem struct {
	fs http.FileSystem
}

func (fs justFilesFilesystem) Open(name string) (http.File, error) {
	f, err := fs.fs.Open(name)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err == nil && stat.IsDir() {
		return nil, os.ErrPermission
	}

	return f, nil
}

// registerStaticWeb 根据 GVA_STATIC_ROOT 开启低内存静态部署，让 Go 后端直接托管 web/dist。
func registerStaticWeb(Router *gin.Engine) {
	// 未设置静态目录时保持原有开发/接口模式，不影响已有 API 路由。
	staticRoot := os.Getenv("GVA_STATIC_ROOT")
	if staticRoot == "" {
		return
	}

	// 统一转成绝对路径，后续做目录边界校验时更可靠。
	staticRoot, err := filepath.Abs(staticRoot)
	if err != nil {
		return
	}

	// index.html 是前端静态部署入口，不存在就跳过静态托管。
	indexPath := filepath.Join(staticRoot, "index.html")
	if !regularFileExists(indexPath) {
		return
	}

	// 托管 Vite 构建后的 JS/CSS/图片等静态资源目录。
	if dirExists(filepath.Join(staticRoot, "assets")) {
		Router.Static("/assets", filepath.Join(staticRoot, "assets"))
	}
	// public/docs 会被 Vite 复制到 dist/docs，这里单独暴露文档资源。
	if dirExists(filepath.Join(staticRoot, "docs")) {
		Router.Static("/docs", filepath.Join(staticRoot, "docs"))
	}

	// 托管 public 目录复制出来的常用根路径文件。
	for _, fileName := range []string{"favicon.ico", "logo.png"} {
		filePath := filepath.Join(staticRoot, fileName)
		if regularFileExists(filePath) {
			Router.StaticFile("/"+fileName, filePath)
		}
	}

	// 根路径直接返回前端入口页。
	Router.GET("/", func(c *gin.Context) {
		c.File(indexPath)
	})

	// 未命中后端 API 或静态文件时，回退到 index.html，支持 Vue hash 路由刷新。
	Router.NoRoute(func(c *gin.Context) {
		// 非 GET/HEAD 请求不做前端回退，避免吞掉错误的 API 调用。
		if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
			c.Status(http.StatusNotFound)
			return
		}

		// 清理请求路径，避免 ../ 这类路径穿越。
		requestPath := filepath.Clean(strings.TrimPrefix(c.Request.URL.Path, "/"))
		if requestPath != "." {
			filePath := filepath.Join(staticRoot, requestPath)
			// 只允许返回静态根目录内的真实文件。
			if pathInsideRoot(staticRoot, filePath) && regularFileExists(filePath) {
				c.File(filePath)
				return
			}
		}

		c.File(indexPath)
	})
}

// regularFileExists 判断路径是否存在且是普通文件。
func regularFileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// dirExists 判断路径是否存在且是目录。
func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// pathInsideRoot 防止静态文件请求逃逸到 GVA_STATIC_ROOT 之外。
func pathInsideRoot(root string, path string) bool {
	relativePath, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return relativePath != ".." && !strings.HasPrefix(relativePath, ".."+string(os.PathSeparator))
}

// 初始化总路由

func Routers() *gin.Engine {
	Router := gin.New()
	// 使用自定义的 Recovery 中间件，记录 panic 并入库
	Router.Use(middleware.GinRecovery(true))
	if gin.Mode() == gin.DebugMode {
		Router.Use(gin.Logger())
	}

	systemRouter := router.RouterGroupApp.System
	exampleRouter := router.RouterGroupApp.Example
	// 如果想要不使用nginx代理前端网页，可以修改 web/.env.production 下的
	// VUE_APP_BASE_API = /
	// VUE_APP_BASE_PATH = http://localhost
	// 然后执行打包命令 npm run build。在打开下面3行注释
	// Router.StaticFile("/favicon.ico", "./dist/favicon.ico")
	// Router.Static("/assets", "./dist/assets")   // dist里面的静态资源
	// Router.StaticFile("/", "./dist/index.html") // 前端网页入口页面

	Router.StaticFS(global.GVA_CONFIG.Local.StorePath, justFilesFilesystem{http.Dir(global.GVA_CONFIG.Local.StorePath)})
	// Router.Use(middleware.LoadTls())  // 如果需要使用https 请打开此中间件 然后前往 core/server.go 将启动模式 更变为 Router.RunTLS("端口","你的cre/pem文件","你的key文件")
	// 跨域，如需跨域可以打开下面的注释
	// Router.Use(middleware.Cors()) // 直接放行全部跨域请求
	// Router.Use(middleware.CorsByRules()) // 按照配置的规则放行跨域请求
	// global.GVA_LOG.Info("use middleware cors")
	docs.SwaggerInfo.BasePath = global.GVA_CONFIG.System.RouterPrefix
	Router.GET(global.GVA_CONFIG.System.RouterPrefix+"/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	global.GVA_LOG.Info("register swagger handler")
	// 方便统一添加路由组前缀 多服务器上线使用

	PublicGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)
	PrivateGroup := Router.Group(global.GVA_CONFIG.System.RouterPrefix)

	PrivateGroup.Use(middleware.JWTAuth()).Use(middleware.CasbinHandler())

	{
		// 健康监测
		PublicGroup.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, "ok")
		})
	}
	{
		systemRouter.InitBaseRouter(PublicGroup) // 注册基础功能路由 不做鉴权
		systemRouter.InitInitRouter(PublicGroup) // 自动初始化相关
	}

	{
		systemRouter.InitApiRouter(PrivateGroup, PublicGroup)               // 注册功能api路由
		systemRouter.InitJwtRouter(PrivateGroup)                            // jwt相关路由
		systemRouter.InitUserRouter(PrivateGroup)                           // 注册用户路由
		systemRouter.InitMenuRouter(PrivateGroup)                           // 注册menu路由
		systemRouter.InitSystemRouter(PrivateGroup)                         // system相关路由
		systemRouter.InitSysVersionRouter(PrivateGroup)                     // 发版相关路由
		systemRouter.InitCasbinRouter(PrivateGroup)                         // 权限相关路由
		systemRouter.InitAutoCodeRouter(PrivateGroup, PublicGroup)          // 创建自动化代码
		systemRouter.InitAuthorityRouter(PrivateGroup)                      // 注册角色路由
		systemRouter.InitSysDictionaryRouter(PrivateGroup)                  // 字典管理
		systemRouter.InitAutoCodeHistoryRouter(PrivateGroup)                // 自动化代码历史
		systemRouter.InitSysOperationRecordRouter(PrivateGroup)             // 操作记录
		systemRouter.InitSysDictionaryDetailRouter(PrivateGroup)            // 字典详情管理
		systemRouter.InitAuthorityBtnRouterRouter(PrivateGroup)             // 按钮权限管理
		systemRouter.InitSysExportTemplateRouter(PrivateGroup, PublicGroup) // 导出模板
		systemRouter.InitSysParamsRouter(PrivateGroup, PublicGroup)         // 参数管理
		systemRouter.InitSysErrorRouter(PrivateGroup, PublicGroup)          // 错误日志
		systemRouter.InitLoginLogRouter(PrivateGroup)                       // 登录日志
		systemRouter.InitApiTokenRouter(PrivateGroup)                       // apiToken签发
		systemRouter.InitSkillsRouter(PrivateGroup, PublicGroup)            // Skills 定义器
		exampleRouter.InitCustomerRouter(PrivateGroup)                      // 客户路由
		exampleRouter.InitFileUploadAndDownloadRouter(PrivateGroup)         // 文件上传下载功能路由
		exampleRouter.InitAttachmentCategoryRouterRouter(PrivateGroup)      // 文件上传下载分类
	}

	//插件路由安装
	InstallPlugin(PrivateGroup, PublicGroup, Router)

	// 注册业务路由
	initBizRouter(PrivateGroup, PublicGroup)

	// API 路由全部注册后再挂静态前端，避免前端回退抢占后端接口。
	registerStaticWeb(Router)

	global.GVA_ROUTERS = Router.Routes()

	global.GVA_LOG.Info("router register success")
	return Router
}
