package main

import "fmt"

func printHelp() {
	fmt.Println(`
_____/\\\\\\\\\\\____/\\\________/\\\__/\\\\\\\\\\\____/\\\\\\\\\___________/\\\\\______        
 ___/\\\/////////\\\_\/\\\_______\/\\\_\/////\\\///___/\\\///////\\\_______/\\\///\\\____       
  __\//\\\______\///__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\/\\\_____/\\\/__\///\\\__      
   ___\////\\\_________\/\\\\\\\\\\\\\\\_____\/\\\_____\/\\\\\\\\\\\/_____/\\\______\//\\\_     
    ______\////\\\______\/\\\/////////\\\_____\/\\\_____\/\\\//////\\\____\/\\\_______\/\\\_    
     _________\////\\\___\/\\\_______\/\\\_____\/\\\_____\/\\\____\//\\\___\//\\\______/\\\__   
      __/\\\______\//\\\__\/\\\_______\/\\\_____\/\\\_____\/\\\_____\//\\\___\///\\\__/\\\____  
       _\///\\\\\\\\\\\/___\/\\\_______\/\\\__/\\\\\\\\\\\_\/\\\______\//\\\____\///\\\\\/_____ 
        ___\///////////_____\///________\///__\///////////__\///________\///_______\/////_______
`)
	fmt.Println("Shiro - AI-Native CI Workflow Runtime")
	fmt.Println()
	fmt.Println("Created by: Rajit Kuthiala")
	fmt.Println("https://www.linkedin.com/in/rajitkuthiala/")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  shiro <command> [options]")
	fmt.Println()
	fmt.Println("Commands:")
	fmt.Println("  build              Build the shiro binary")
	fmt.Println("  test               Run tests")
	fmt.Println("  run <workflow>     Run a workflow (default if no command specified)")
	fmt.Println("  help               Show this help message")
	fmt.Println()
	fmt.Println("Run Command Options:")
	fmt.Println("  -workflow <file>   Path to workflow JSON file (required)")
	fmt.Println("  -config <file>     Path to model configuration file (default: configs/models.yaml)")
	fmt.Println("  -state-store <type> State store type: memory, filesystem, gitlab (default: gitlab)")
	fmt.Println("  -help              Show help for run command")
	fmt.Println()
	fmt.Println("Examples:")
	fmt.Println("  shiro build")
	fmt.Println("  shiro test")
	fmt.Println("  shiro run examples/print-example.json")
	fmt.Println("  shiro examples/mr-review.json -config configs/models.yaml")
	fmt.Println()
	fmt.Println("For more information, visit: https://github.com/rajitk13/shiro-automation")
}

func printRunHelp() {
	fmt.Println(`
   ____  _   _  ____  _   _ ___ _   _  ____ 
  / ___|| \ | ||  _ \| \ | |_ _| \ | |/ ___|
 | |    |  \| || | | |  \| || ||  \| | |  _ 
 | |___ | |\  || |_| | |\  || || |\  | |_| |
  \____||_| \_||____/|_| \_||___|_| \_|\____|
`)
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
