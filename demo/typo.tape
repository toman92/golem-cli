Output demo/typo.gif
Set FontSize 20
Set Width 1200
Set Height 800
Set Padding 40

Type "./golem 'copy all .txt files from demo/fixtures/projects to demo/fixtures/project-dump'"
Sleep 500ms
Enter

Sleep 3s

# Typo fuzzy match (project-dump -> projects-dump)
Type "y"
Sleep 500ms
Enter

# Confirm copy
Sleep 2s
Type "n"
Sleep 500ms
Enter

Sleep 3s
