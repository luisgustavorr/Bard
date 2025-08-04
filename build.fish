#!/usr/bin/fish
go build -o bard main.go
mv bard ~/bin/  # ou outro diretório no PATH
bard completion fish > ~/.config/fish/completions/bard.fish
. ~/.config/fish/completions/bard.fish 