# vivecaka README demo
#
# Two-pane tmux layout:
#   Left  (pane 0): "repo pulse"  — baseline GitHub CLI view of anomalyco/opencode
#   Right (pane 1): "vivecaka"    — terminal-native PR triage, detail, and diff flow

Output assets/demo.gif
Set Shell bash
Set FontSize 15
Set FontFamily "Menlo"
Set Width 1600
Set Height 900
Set Padding 20
Set Theme "Catppuccin Mocha"
Set TypingSpeed 40ms
Set CursorBlink false
Set Framerate 30
Set PlaybackSpeed 1.0

Require tmux
Require gh

# Hidden setup
Hide

Type "make build >/dev/null 2>&1"
Enter
Sleep 3s

Type "tmux -L vivecaka-demo kill-server 2>/dev/null"
Enter
Sleep 300ms

Type "tmux -2 -f /dev/null -L vivecaka-demo new-session -d -s demo 'bash --noprofile --init-file ./scripts/demo-init-left.sh'"
Enter
Sleep 500ms

Type "tmux -L vivecaka-demo split-window -h -t demo 'bash --noprofile --init-file ./scripts/demo-init-right.sh'"
Enter
Sleep 500ms

Type 'tmux -L vivecaka-demo set -g default-terminal "tmux-256color"'
Enter
Sleep 100ms
Type 'tmux -L vivecaka-demo set -as terminal-features ",*:RGB"'
Enter
Sleep 100ms
Type 'tmux -L vivecaka-demo set -t demo pane-border-style "fg=#313244"'
Enter
Sleep 100ms
Type 'tmux -L vivecaka-demo set -t demo pane-active-border-style "fg=#89b4fa"'
Enter
Sleep 100ms
Type "tmux -L vivecaka-demo set -t demo pane-border-status top"
Enter
Sleep 100ms
Type 'tmux -L vivecaka-demo set -t demo pane-border-format " #{?#{==:#{pane_index},0},repo pulse,vivecaka} "'
Enter
Sleep 100ms

Type 'tmux -L vivecaka-demo set -t demo status-style "bg=#11111b,fg=#cdd6f4"'
Enter
Sleep 100ms
Type `tmux -L vivecaka-demo set -t demo status-left '#[bg=#cba6f7,fg=#11111b,bold] vivecaka #[bg=#11111b] '`
Enter
Sleep 100ms
Type `tmux -L vivecaka-demo set -t demo status-right '#[fg=#9399b2]terminal-first pr review #[bg=#89b4fa,fg=#11111b,bold] LIVE '`
Enter
Sleep 100ms
Type "tmux -L vivecaka-demo set -t demo status-left-length 20"
Enter
Sleep 100ms
Type "tmux -L vivecaka-demo set -t demo status-right-length 48"
Enter
Sleep 100ms
Type `tmux -L vivecaka-demo setw -t demo window-status-format ''`
Enter
Sleep 100ms
Type `tmux -L vivecaka-demo setw -t demo window-status-current-format ''`
Enter
Sleep 100ms

Type "tmux -L vivecaka-demo select-pane -t demo.0"
Enter
Sleep 200ms

Type "clear"
Enter
Sleep 200ms

Show

# Recording starts here
Type "tmux -2 -L vivecaka-demo attach -t demo"
Enter
Sleep 1s

# Left pane: show the raw GitHub PR queue first
Type "gh pr list -R anomalyco/opencode --limit 8"
Enter
Sleep 2500ms

# Right pane: launch vivecaka against the same repo
Ctrl+B
Right
Sleep 500ms

Type "./bin/vivecaka --repo anomalyco/opencode"
Enter
Sleep 6s

# Open the currently selected PR
Enter
Sleep 2500ms

# Jump to the files tab
Type "3"
Sleep 1800ms

# Open the diff and show split mode
Type "d"
Sleep 2500ms
Type "t"
Sleep 2200ms

# Small amount of navigation to make the UI feel alive
Type "]"
Sleep 1200ms
Type "}"
Sleep 1200ms
