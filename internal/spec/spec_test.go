package spec

import (
	"fmt"
	"os"
	"testing"
	// "gopkg.in/yaml.v3"
)

func TestParseGemMetadata(t *testing.T) {
	res, err := os.ReadFile("../../test/devise-metadata.yml")
	if err != nil {
		panic(err)
	}
	metadata := ParseGemMetadata([]byte(res))
	fmt.Println(metadata.Emails)
	// fmt.Println(fmt.Sprintf("%T", metadata.Dependencies[1].Requirement.Requirements))
	// var arg string
	// var ver string
	// for _, entry := range metadata.Dependencies[1].Requirement.Requirements {
	//      switch i := entry.(type) {
	//      case []interface{}: {
	//      	for _, lol := range i {
	//      		if fmt.Sprintf("%T", lol) == "string" {
	//      		  arg = fmt.Sprintf("%s", lol)
	//      	  } else {
	//      	  	mymap := lol.(map[string]interface{})
	//      	  	ver = fmt.Sprintf("%s", mymap["version"])
	//      	  }
	//      	}
	//      }
	//      default:
	//          fmt.Printf("Type i=%T\n", i)
	//      }
	//  }
	//  fmt.Println("arg=", arg)
	//  fmt.Println("version=", ver)
}
