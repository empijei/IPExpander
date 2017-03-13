grep -Po "//DOT \K.*" parsers/dashed.go | 
dot -Tpng > 
parsers/automata.png
