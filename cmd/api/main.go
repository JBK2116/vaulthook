// Package main is the entry point for the vaulthook application.
package main

import "github.com/JBK2116/vaulthook/internal/app"

func main() {
	application := app.New()
	application.Run()
}
