package main

import (
	"os"

	"github.com/rkuthiala/shiro-automation/internal/cli"
)

func main() {
	if len(os.Args) < 2 {
		printHelp()
		os.Exit(1)
	}

	command := os.Args[1]
	args := os.Args[2:]

	switch command {
	case "init":
		cli.InitCommand(args)
	case "run":
		cli.RunCommand(args)
	case "add":
		cli.ModuleCommand(args)
	case "search":
		cli.ModuleCommand(args)
	case "list":
		cli.ModuleCommand(args)
	case "remove":
		cli.ModuleCommand(args)
	case "install":
		cli.ModuleCommand(args)
	case "info":
		cli.ModuleCommand(args)
	case "docs":
		cli.ModuleCommand(args)
	case "module":
		cli.ModuleCommand(args)
	case "approve":
		cli.ApproveCommand(args)
	case "help", "-help", "--help":
		printHelp()
	default:
		// Try to run as workflow if no recognized command
		cli.RunCommand(os.Args[1:])
	}
}

func printHelp() {
	println(`_____/\\\\\\\\\\\____/\\\________/\\\__/\\\\\\\\\\\____/\\\\\\\\\___________/\\\\\______        
 ___/\\\/////////\\\_\/\\\_______\/\\\_\/////\\\///___/\\\///////\\\_______/\\\///\\\____       
  __\//\\\______\///__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\/\\\_____/\\\/__\///\\\__      
   ___\////\\\_________\/\\\\\\\\\\\\\\\_____\/\\\_____\/\\\\\\\\\\\/_____/\\\______\//\\\_     
    ______\////\\\______\/\\\/////////\\\_____\/\\\_____\/\\\//////\\\____\/\\\_______\/\\\_    
     _________\////\\\___\/\\\_______\/\\\_____\/\\\_____\/\\\____\//\\\___\//\\\______/\\\__   
      __/\\\______\//\\\__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\//\\\___\///\\\__/\\\____  
       _\///\\\\\\\\\\\/___\/\\\_______\/\\\__/\\\\\\\\\\\_\/\\\______\//\\\____\///\\\\\/_____ 
        ___\///////////_____\///________\///__\///////////__\///________\///_______\/////_______`)
	println("Shiro Automation - AI-Native CI Workflow Runtime")
	println()
	println("Usage: shiro <command> [options]")
	println()
	println("Commands:")
	println("  init              Initialize a Shiro project")
	println("  run               Run a workflow")
	println("  add module        Add a module")
	println("  list modules      List available modules")
	println("  search module     Search for modules")
	println("  remove module     Remove a module")
	println("  install module    Install a module from GitHub")
	println("  info module       Display module information")
	println("  docs module       Open module documentation")
	println("  approve           Manage workflow approvals")
	println("  help              Show this help message")
	println()
	println("Examples:")
	println("  shiro init")
	println("  shiro run")
	println("  shiro add module jira")
	println("  shiro list modules")
	println("  shiro approve list")
	println()
	println("For more information: https://github.com/rajitk13/shiro-automation")
}
