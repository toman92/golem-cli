Output demo/copy-safe.gif
Set FontSize 20
Set Width 1200
Set Height 800
Set Padding 40

Type "./golem 'copy all .md files from demo/fixtures/projects into a new folder called demo/fixtures/projects-dump/docs'"
Sleep 500ms
Enter

# No escape warnings because both are inside $PWD/demo/fixtures
# Final copy confirmation
Sleep 3s
Type "y"
Sleep 500ms
Enter

Sleep 4s
