package main

import "fmt"

func printHelp() {
	fmt.Println(`_____/\\\\\\\\\\\____/\\\________/\\\__/\\\\\\\\\\\____/\\\\\\\\\___________/\\\\\______        
 ___/\\\/////////\\\_\/\\\_______\/\\\_\/////\\\///___/\\\///////\\\_______/\\\///\\\____       
  __\//\\\______\///__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\/\\\_____/\\\/__\///\\\__      
   ___\////\\\_________\/\\\\\\\\\\\\\\\_____\/\\\_____\/\\\\\\\\\\\/_____/\\\______\//\\\_     
    ______\////\\\______\/\\\/////////\\\_____\/\\\_____\/\\\//////\\\____\/\\\_______\/\\\_    
     _________\////\\\___\/\\\_______\/\\\_____\/\\\_____\/\\\____\//\\\___\//\\\______/\\\__   
      __/\\\______\//\\\__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\//\\\___\///\\\__/\\\____  
       _\///\\\\\\\\\\\/___\/\\\_______\/\\\__/\\\\\\\\\\\_\/\\\______\//\\\____\///\\\\\/_____ 
        ___\///////////_____\///________\///__\///////////__\///________\///_______\/////_______`)
	fmt.Println("Shiro - AI-Native CI Workflow Runtime")
	fmt.Println()
	fmt.Println("Quick Start:")
	fmt.Println("  shiro init                   - Initialize Shiro in your project")
	fmt.Println("  shiro run                    - Run workflow from .shiro/workflow.json")
	fmt.Println("  shiro add module jira        - Add a module")
	fmt.Println("  shiro search module jira     - Search for modules")
	fmt.Println("  shiro list modules           - List installed modules")
	fmt.Println()
	fmt.Println("Project Initialization:")
	fmt.Println("  shiro init                   - Create .shiro folder with example files")
	fmt.Println("                               (workflow, config, module registry)")
	fmt.Println()
	fmt.Println("Module Commands:")
	fmt.Println("  shiro add module <name>      - Add a module (auto-discovers from official repo)")
	fmt.Println("  shiro add module <git-url>   - Add module from GitHub repository")
	fmt.Println("  shiro search module <query>  - Search GitHub for modules")
	fmt.Println("  shiro install module <repo>  - Install module from GitHub")
	fmt.Println("  shiro list modules           - List all installed modules")
	fmt.Println("  shiro remove module <name>   - Remove a module")
	fmt.Println("  shiro info module <name>     - Show module information")
	fmt.Println("  shiro docs module <name>     - Open module documentation")
	fmt.Println()
	fmt.Println("Advanced Options:")
	fmt.Println("  shiro run -workflow <file>   - Run specific workflow file")
	fmt.Println("  shiro run -config <file>     - Use specific config file")
	fmt.Println("  shiro run -shiro-dir <path>  - Use custom .shiro directory")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  # Initialize a new project")
	fmt.Println("  shiro init")
	fmt.Println()
	fmt.Println("  # Quick start - assumes .shiro/workflow.json exists")
	fmt.Println("  shiro run")
	fmt.Println()
	fmt.Println("  # Add official module (auto-discovers)")
	fmt.Println("  shiro add module jira")
	fmt.Println()
	fmt.Println("  # Add custom module from GitHub")
	fmt.Println("  shiro add module github.com/user/custom-module")
	fmt.Println()
	fmt.Println("  # Run with custom directory")
	fmt.Println("  shiro run -shiro-dir /path/to/project/.shiro")
	fmt.Println()
	fmt.Println("For more information: https://github.com/rkuthiala/shiro-automation")
}

func printRunHelp() {
	fmt.Println(`_____/\\\\\\\\\\\____/\\\________/\\\__/\\\\\\\\\\\____/\\\\\\\\\___________/\\\\\______        
 ___/\\\/////////\\\_\/\\\_______\/\\\_\/////\\\///___/\\\///////\\\_______/\\\///\\\____       
  __\//\\\______\///__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\/\\\_____/\\\/__\///\\\__      
   ___\////\\\_________\/\\\\\\\\\\\\\\\_____\/\\\_____\/\\\\\\\\\\\/_____/\\\______\//\\\_     
    ______\////\\\______\/\\\/////////\\\_____\/\\\_____\/\\\//////\\\____\/\\\_______\/\\\_    
     _________\////\\\___\/\\\_______\/\\\_____\/\\\_____\/\\\____\//\\\___\//\\\______/\\\__   
      __/\\\______\//\\\__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\//\\\___\///\\\__/\\\____  
       _\///\\\\\\\\\\\/___\/\\\_______\/\\\__/\\\\\\\\\\\_\/\\\______\//\\\____\///\\\\\/_____ 
        ___\///////////_____\///________\///__\///////////__\///________\///_______\/////_______`)
	fmt.Println("Shiro - AI-Native CI Workflow Runtime")
	fmt.Println()
	fmt.Println("Created by: Rajit Kuthiala")
	fmt.Println("https://www.linkedin.com/in/rajitkuthiala/")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  shiro run [options]")
	fmt.Println("  shiro [options] (shorthand)")
	fmt.Println()
	fmt.Println("Options:")
	fmt.Println("  -workflow <file>   Path to workflow JSON file (required)")
	fmt.Println("  -config <file>     Path to model configuration file (default: configs/models.yaml)")
	fmt.Println("  -state-store <type> State store type: memory, filesystem, gitlab (default: gitlab)")
	fmt.Println("  -help              Show this help message")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro run examples/print-example.json")
	fmt.Println("  shiro run examples/mr-review.json -config configs/models.yaml")
	fmt.Println("  shiro examples/github-mr-review.json -state-store filesystem")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/rajitk13/shiro-automation")
}
