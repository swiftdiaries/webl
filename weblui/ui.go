package main

import (
  "fmt"
  "github.com/swiftdiaries/webl/api"
)

func showVersion() {
  webl.INFO.Println(fmt.Sprintf("weblui %s", webl.Version()))
}
