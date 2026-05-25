package ast

import (
	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"os"
	"path/filepath"
	"testing"
)

func init() {
	global.GVA_CONFIG.AutoCode.Root, _ = filepath.Abs("../../../")
	global.GVA_CONFIG.AutoCode.Server = "server"
}

func TestMain(m *testing.M) {
	root, err := os.MkdirTemp("", "gva-ast-fixtures-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(root)
	if err := writeASTFixtures(root); err != nil {
		panic(err)
	}
	global.GVA_CONFIG.AutoCode.Root = root
	global.GVA_CONFIG.AutoCode.Server = "server"
	os.Exit(m.Run())
}

func writeASTFixtures(root string) error {
	files := map[string]string{
		"server/api/v1/enter.go": `package v1

type ApiGroup struct {
	ExampleApiGroup example.ApiGroup
}
`,
		"server/router/enter.go": `package router

type RouterGroup struct {
	Example example.RouterGroup
}
`,
		"server/service/enter.go": `package service

type ServiceGroup struct {
	ExampleServiceGroup example.ServiceGroup
}
`,
		"server/router/example/enter.go": `package example

type RouterGroup struct {
	FileUploadAndDownloadRouter
}

var exaFileUploadAndDownloadApi = api.ApiGroupApp.ExampleApiGroup.FileUploadAndDownloadApi
`,
		"server/api/v1/example/enter.go": `package example

type ApiGroup struct {
	FileUploadAndDownloadApi
}

var fileUploadAndDownloadService = service.ServiceGroupApp.ExampleServiceGroup.FileUploadAndDownloadService
`,
		"server/service/example/enter.go": `package example

type ServiceGroup struct {
	FileUploadAndDownloadService
}
`,
		"server/initialize/gorm_biz.go": `package initialize

func bizModel() error {
	db := global.GVA_DB
	return db.AutoMigrate(&example.ExaFileUploadAndDownload{}, &example.ExaCustomer{})
}
`,
		"server/initialize/router_biz.go": `package initialize

func initBizRouter(privateGroup, publicGroup any) {
	{
		exampleRouter := router.RouterGroupApp.Example
		exampleRouter.InitCustomerRouter(privateGroup, publicGroup)
		exampleRouter.InitFileUploadAndDownloadRouter(privateGroup, publicGroup)
	}
}
`,
		"server/plugin/register.go": `package plugin

import (
	_ "github.com/flipped-aurora/gin-vue-admin/server/plugin/existing"
)
`,
		"server/plugin/gva/plugin.go": `package gva

type Plugin struct{}
`,
		"server/plugin/gva/api/enter.go": `package api

type ApiGroup struct {
	User user
}

var serviceUser = service.Service.User
`,
		"server/plugin/gva/router/enter.go": `package router

type RouterGroup struct {
	User user
}

var userApi = api.Api.User
`,
		"server/plugin/gva/service/enter.go": `package service

type ServiceGroup struct {
	User user
}
`,
		"server/plugin/gva/gen/main.go": `package main

func main() {
	gen.ApplyBasic(model.User{})
}
`,
		"server/plugin/gva/initialize/gorm.go": `package initialize

func Gorm(db DB) error {
	return db.AutoMigrate(&model.User{}, &model.SysUser{})
}
`,
		"server/plugin/gva/initialize/router.go": `package initialize

func Router(public, private any) {
	router.Router.User.Init(public, private)
}
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
