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
	case "validate":
		cli.ValidateCommand(args)
	case "set":
		cli.SetCommand(args)
	case "get":
		cli.GetCommand(args)
	case "delete":
		cli.DeleteCommand(args)
	case "add":
		cli.ModuleCommand(append([]string{"add"}, args...))
	case "search":
		cli.ModuleCommand(append([]string{"search"}, args...))
	case "list":
		if len(args) > 0 && args[0] == "modules" {
			cli.ModuleCommand(append([]string{"list"}, args...))
		} else {
			cli.ListCommand(args)
		}
	case "remove":
		cli.ModuleCommand(append([]string{"remove"}, args...))
	case "install":
		cli.ModuleCommand(append([]string{"install"}, args...))
	case "info":
		cli.ModuleCommand(append([]string{"info"}, args...))
	case "docs":
		cli.ModuleCommand(append([]string{"docs"}, args...))
	case "module":
		cli.ModuleCommand(args)
	case "build":
		cli.BuildCommand(args)
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
	println("  validate          Validate a workflow")
	println("  build             Build shiro with all modules")
	println("  set <key> <val>   Store a value in data store")
	println("  get <key>         Retrieve a value from data store")
	println("  delete <key>      Delete a value from data store")
	println("  list              List data store keys")
	println("  add module        Add a module")
	println("  list modules      List available modules")
	println("  search module     Search for modules")
	println("  remove module     Remove a module")
	println("  install module    Install a module from GitHub")
	println("  info module       Display module information")
	println("  docs module       Open module documentation")
	println("  help              Show this help message")
	println()
	println("Examples:")
	println("  shiro init")
	println("  shiro run")
	println("  shiro validate -workflow .shiro/workflow.json")
	println("  shiro build")
	println("  shiro set build_id 12345")
	println("  shiro get build_id")
	println("  shiro add module jira")
	println("  shiro list modules")
	println()
	println("For more information: https://github.com/rajitk13/shiro-automation")
}
