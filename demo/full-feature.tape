Output demo/full-feature.gif
Set FontSize 20
Set Width 1200
Set Height 800
Set Padding 40

# Start Interactive Shell
Type "./golem"
Sleep 500ms
Enter
Sleep 1s

# Demo 1: Safe Copy with Typo Auto-Correct
Type "copy all .md files from demo/fixtures/projects to demo/fixtures/project-dump/docs"
Sleep 500ms
Enter
Sleep 2s

# Auto-correct typo to projects-dump
Type "y"
Sleep 500ms
Enter
Sleep 1s

# Confirm copy
Type "y"
Sleep 500ms
Enter
Sleep 2s

# Demo 2: Sandbox Escape Move
Type "move all .txt files from demo/fixtures/projects to /tmp/golem-demo-escape"
Sleep 500ms
Enter
Sleep 3s

# Allow Escape
Type "y"
Sleep 500ms
Enter
Sleep 1s

# Confirm Move
Type "y"
Sleep 500ms
Enter
Sleep 1s

# Demo 3: Safe Delete (Whitelist)
Type "delete the build folder in demo/fixtures/projects"
Sleep 500ms
Enter
Sleep 2s

# Confirm Delete
Type "y"
Sleep 500ms
Enter
Sleep 1s

# Show Model Configuration
Type "/config model"
Sleep 500ms
Enter
Sleep 1s
Type "1"
Sleep 500ms
Enter
Sleep 1s

# Exit
Type "/exit"
Sleep 500ms
Enter
Sleep 2s
