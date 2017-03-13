# IPExpander
If you are looking for a tool to automatically expand IPs of a subnet you can probably find a more updated and reliable version [here](https://github.com/AnnaOpss/IPExpander)

# So, what is this then?
This is just an exercise of writing a lexer/parser for a formal language from scratch

## Dashed parser:
### Definitions
This is the lexer automata for the language:
![alt text](/parsers/automata.png "State Machine")

Terminal states are double-circled. All states can lead to an error.

The emitted tokens are marked by the symbol "→"

Token begin cursor moves are marked by "move cursor"

### Samples
Some examples of the language are:
```
	"10.0.0.1-2"
	"10.0.1-2.1"
	"10-11.0.1-2.1"
```

It also accepts missing extremes, 0 or 255 will be used if not specified
```
	"10.0.1-2.254-"
	"10.0.1-2.255-"
```

For example the following IP definition:
```
	"-1.0.-.1"
```

expands to:
```
0.0.255.1
1.0.255.1
```

The final output can also wrap around limits:
```
	"10.0.0.255-1"
```

which expands to:
```
10.0.0.255
10.0.0.0
10.0.0.1
```
### Output
The final output of the parser is a `4x2` matrix containing the ranges the IPs should vary within (extremes are included) that is then converted into the list of `net.IP` addresses

# Why
The parsing approach used is described in [Lexical Scanning in Go](https://www.youtube.com/watch?v=HxaD_trXwRE) by Rob Pike, I just wanted to try it out with minor modifications and big simplifications in order to make it work in this specific case.
