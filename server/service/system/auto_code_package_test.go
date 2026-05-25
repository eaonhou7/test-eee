package system

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	model "github.com/flipped-aurora/gin-vue-admin/server/model/system"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func Test_autoCodePackage_Create(t *testing.T) {
	setupAutoCodePackageTest(t)

	type args struct {
		ctx  context.Context
		info *request.SysAutoCodePackageCreate
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "测试 package",
			args: args{
				ctx: context.Background(),
				info: &request.SysAutoCodePackageCreate{
					Template:    "package",
					PackageName: "demo",
				},
			},
			wantErr: false,
		},
		{
			name: "测试 plugin",
			args: args{
				ctx: context.Background(),
				info: &request.SysAutoCodePackageCreate{
					Template:    "plugin",
					PackageName: "plugdemo",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &autoCodePackage{}
			if err := a.Create(tt.args.ctx, tt.args.info); (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_autoCodePackage_templates(t *testing.T) {
	setupAutoCodePackageTest(t)

	type args struct {
		ctx       context.Context
		entity    model.SysAutoCodePackage
		info      request.AutoCode
		isPackage bool
	}
	tests := []struct {
		name      string
		args      args
		wantCode  map[string]string
		wantEnter map[string]map[string]string
		wantErr   bool
	}{
		{
			name: "测试1",
			args: args{
				ctx: context.Background(),
				entity: model.SysAutoCodePackage{
					Desc:        "描述",
					Label:       "展示名",
					Template:    "plugin",
					PackageName: "preview",
				},
				info: request.AutoCode{
					Abbreviation:    "user",
					GenerateServer:  true,
					HumpPackageName: "user",
				},
				isPackage: false,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &autoCodePackage{}
			gotCode, gotEnter, gotCreates, err := s.templates(tt.args.ctx, tt.args.entity, tt.args.info, tt.args.isPackage)
			if (err != nil) != tt.wantErr {
				t.Errorf("templates() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			for key, value := range gotCode {
				t.Logf("\n")
				t.Logf("%s", key)
				t.Logf("%s", value)
				t.Logf("\n")
			}
			t.Log(gotCreates)
			if gotCode == nil || gotEnter == nil || gotCreates == nil {
				t.Fatalf("templates() returned nil maps: code=%v enter=%v creates=%v", gotCode, gotEnter, gotCreates)
			}
			if len(gotCreates) == 0 {
				t.Fatalf("templates() expected create targets")
			}
		})
	}
}

func setupAutoCodePackageTest(t *testing.T) {
	t.Helper()

	previousAutoCode := global.GVA_CONFIG.AutoCode
	previousDB := global.GVA_DB
	root := t.TempDir()
	global.GVA_CONFIG.AutoCode.Root = root
	global.GVA_CONFIG.AutoCode.Server = "server"
	global.GVA_CONFIG.AutoCode.Web = "web"
	global.GVA_CONFIG.AutoCode.Module = "github.com/flipped-aurora/gin-vue-admin/server"

	if err := writeAutoCodePackageFixtures(root); err != nil {
		t.Fatalf("write autocode fixtures: %v", err)
	}

	db, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "autocode.db")), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	if err := db.AutoMigrate(&model.SysAutoCodePackage{}, &model.SysAutoCodeHistory{}); err != nil {
		t.Fatalf("migrate autocode package table: %v", err)
	}
	global.GVA_DB = db

	t.Cleanup(func() {
		global.GVA_CONFIG.AutoCode = previousAutoCode
		global.GVA_DB = previousDB
	})
}

func writeAutoCodePackageFixtures(root string) error {
	files := map[string]string{
		"server/api/v1/enter.go": `package v1

type ApiGroup struct{}
`,
		"server/router/enter.go": `package router

type RouterGroup struct{}
`,
		"server/service/enter.go": `package service

type ServiceGroup struct{}
`,
		"server/plugin/register.go": `package plugin
`,
		"server/resource/package/server/api/enter.go.tpl": `package {{ .Package }}

type ApiGroup struct{}
`,
		"server/resource/package/server/router/enter.go.tpl": `package {{ .Package }}

type RouterGroup struct{}
`,
		"server/resource/package/server/service/enter.go.tpl": `package {{ .Package }}

type ServiceGroup struct{}
`,
		"server/resource/package/server/model/model.go.tpl": `package {{ .Package }}

type {{ .StructName }} struct{}
`,
		"server/resource/plugin/server/plugin.go.tpl": `package {{ .Package }}

type Plugin struct{}
`,
		"server/resource/plugin/server/model/model.go.tpl": `package model

type {{ .StructName }} struct{}
`,
	}
	for name, body := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(body), 0644); err != nil {
			return err
		}
	}
	return nil
}

func seedAutoCodePackage(t *testing.T, packageName, template string) {
	t.Helper()
	entity := model.SysAutoCodePackage{
		PackageName: packageName,
		Template:    template,
		Module:      global.GVA_CONFIG.AutoCode.Module,
	}
	if err := global.GVA_DB.Create(&entity).Error; err != nil {
		t.Fatalf("seed autocode package %s/%s: %v", packageName, template, err)
	}
}
