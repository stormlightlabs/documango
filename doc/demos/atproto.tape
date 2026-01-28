# VHS tape for documango atproto demo
# Run with: vhs < doc/demo.tape

Output demo.mp4

Set Shell "zsh"
Set FontSize 16
Set FontFamily "GeistMono Nerd Font Mono"
Set Width 1200
Set Height 800
Set Padding 20
Set Theme "Catppuccin Mocha"

Sleep 1s
Type "clear"
Enter
Sleep 500ms

Type "./tmp/documango search pds"
Enter
Sleep 4s

Type "./tmp/documango search 'feed post'"
Enter
Sleep 2s

Type "./tmp/documango info atproto/lexicon/app.bsky.feed.post"
Enter
Sleep 3s

Type "./tmp/documango read -p -r -w 100 atproto/lexicon/app.bsky.feed.post"
Enter
Sleep 5s

Type "./tmp/documango search -t Spec 'sync'"
Enter
Sleep 2s

Type "./tmp/documango read -p -r -w 100 atproto/spec/sync"
Enter
Sleep 5s
