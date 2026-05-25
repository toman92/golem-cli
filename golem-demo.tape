Output golem-demo.gif
Set FontSize 20
Set Width 1200
Set Height 800
Set Padding 40

Type "./golem"
Sleep 500ms
Enter

Sleep 2s
Type "copy all .md files, exlucding README.md files, from my dev/personal folder into a new folder called project-dump/docs in my dev folder"
Sleep 500ms
Enter

# Wait for Ollama to process the plan
Sleep 2s

# Source path escape confirmation
Type "y"
Sleep 500ms
Enter

# Destination path escape confirmation
Sleep 2s
Type "y"
Sleep 500ms
Enter

# Final copy confirmation
Sleep 3s
Type "n"
Sleep 500ms
Enter

Sleep 1s
Type "/exit"
Sleep 500ms
Enter

Sleep 1s
Type "./golem 'copy all README.md files from my dev/personal folder into a new folder called projects-dump in my dev folder'"
Sleep 500ms
Enter

# Wait for Ollama
Sleep 2s

# Source escape
Type "y"
Sleep 500ms
Enter

# Dest escape
Sleep 2s
Type "y"
Sleep 500ms
Enter

# Confirm copy
Sleep 3s
Type "y"
Sleep 500ms
Enter

# Wait to show success logs before GIF ends
Sleep 5s