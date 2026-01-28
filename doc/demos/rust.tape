# VHS tape for documango rust demo
# Run with: vhs < doc/demos/rust.tape

Output doc/demos/rust.mp4

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

Type "./tmp/documango search serde"
Enter
Sleep 4s

Type "./tmp/documango search 'clap ValueEnum'"
Enter
Sleep 2s

Type "./tmp/documango info rust/serde/de"
Enter
Sleep 3s

Type "./tmp/documango read -p -r -w 100 rust/clap/ValueEnum"
Enter
Sleep 5s

Type "./tmp/documango search 'serde deserialize'"
Enter
Sleep 2s

Type "./tmp/documango read -p -r -w 100 rust/serde/de"
Enter
Sleep 5s
