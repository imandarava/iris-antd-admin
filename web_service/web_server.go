package web_server

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"iris-antd-admin/libs"
	"iris-antd-admin/models"
	"iris-antd-admin/routes"
	"strings"
	"time"

	"github.com/kataras/iris/v12/context"
)

type Server struct {
	App *iris.Application
	//AssetFile	  http.FileSystem
	//Asset      func(name string) ([]byte, error)
	//AssetNames func() []string
}

func NewServer() *Server {
	app := iris.Default()
	return &Server{
		App: app,
		//AssetFile:  assetFile,
		//Asset:      asset,
		//AssetNames: assetNames,
	}
}

func (s *Server) Serve() error {
	if libs.Config.HTTPS {
		host := fmt.Sprintf("%s:%d", libs.Config.Host, 443)
		if err := s.App.Run(iris.TLS(host, libs.Config.Certpath, libs.Config.Certkey)); err != nil {
			return err
		}
	} else {
		if err := s.App.Run(
			iris.Addr(fmt.Sprintf("%s:%d", libs.Config.Host, libs.Config.Port)),
			iris.WithoutServerError(iris.ErrServerClosed),
			iris.WithOptimizations,
			iris.WithTimeFormat(time.RFC3339),
		); err != nil {
			return err
		}
	}

	return nil
}

func (s *Server) NewApp() {
	s.App.Logger().SetLevel(libs.Config.LogLevel)

	libs.InitCasbin()
	libs.InitDb()

	db := libs.Db
	db.AutoMigrate(
		&models.User{},
		&models.Company{},
		&models.OauthToken{},
		&models.Role{},
		&models.Permission{},
	)

	iris.RegisterOnInterrupt(func() {
		sqlDB, _ := db.DB()
		_ = sqlDB.Close()
	})

	routes.App(s.App) //注册 app 路由
}

type PathName struct {
	Name   string
	Path   string
	Method string
}

// 获取路由信息
func (s *Server) GetRoutes() []*models.Permission {
	var rrs []*models.Permission
	names := getPathNames(s.App.GetRoutesReadOnly())
	if libs.Config.Debug {
		fmt.Println(fmt.Sprintf("路由权限集合：%v", names))
		fmt.Println(fmt.Sprintf("Iris App ：%v", s.App))
	}
	for _, pathName := range names {
		if isPermRoute(pathName.Method) {
			rr := &models.Permission{Name: pathName.Path, DisplayName: pathName.Name, Description: pathName.Name, Act: pathName.Method}
			rrs = append(rrs, rr)
		}
	}
	return rrs
}

func getPathNames(routeReadOnly []context.RouteReadOnly) []*PathName {
	var pns []*PathName
	if libs.Config.Debug {
		fmt.Println(fmt.Sprintf("routeReadOnly：%v", routeReadOnly))
	}
	for _, s := range routeReadOnly {
		pn := &PathName{
			Name:   s.Name(),
			Path:   s.Path(),
			Method: s.Method(),
		}
		pns = append(pns, pn)
	}

	return pns
}

// 过滤非必要权限
func isPermRoute(name string) bool {
	exceptRouteName := []string{"OPTIONS", "GET", "POST", "DELETE", "HEAD", "PUT", "PATCH", "payload"}
	for _, er := range exceptRouteName {
		if strings.Contains(name, er) {
			return true
		}
	}
	return false
}
