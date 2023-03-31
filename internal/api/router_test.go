package api

// import (
// 	"github.com/gemfast/server/internal/config"
// 	"github.com/gemfast/server/internal/db"
// 	"testing"
// )

// func TestInitRouterNoneAuth(t *testing.T) {
// 	config.InitConfig()
// 	config.Env.AuthMode = "none"
// 	r := initRouter()
// 	for _, route := range r.Routes() {
// 		if route.Path == "/admin/login" {
// 			t.Errorf("NoneAuth: expected none auth handler to be configured")
// 		}
// 	}
// }

// func TestInitRouterLocalAuth(t *testing.T) {
// 	config.InitConfig()
// 	config.Env.AuthMode = "local"
// 	err := db.Connect()
// 	if err != nil {
// 		t.Errorf("LocalAuth: unable to initdb")
// 	}
// 	defer db.BoltDB.Close()
// 	r := initRouter()
// 	var match bool
// 	for _, route := range r.Routes() {
// 		if route.Path == "/admin/login" {
// 			match = true
// 			break
// 		}
// 	}
// 	if !match {
// 		t.Errorf("LocalAuth: expected local auth handler to be configured")
// 	}
// }
