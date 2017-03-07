package parsers

import (
	"fmt"
	"io"
	"net"
	"strconv"
	"unicode"
)

type State func(Input, chan byte) (State, error)

type Input struct {
	input       string
	token_begin int
	current     int
}

type outIPv4 struct {
	out    [4][2]byte
	cursor int
}

func (o *outIPv4) Write(p byte) error {
	if o.cursor > 7 {
		return io.EOF
	}
	o.out[int(o.cursor/2)][o.cursor%2] = p
	o.cursor++
	return nil
}

func ParseDashed(in string) (out []net.IP, err error) {
	source := Input{in, 0, 0}
	ranges, err := parse(source)
	if err != nil {
		return nil, err
	}
	var iterRanges [4][2]int
	for i, se := range ranges {
		//User specified an overflowing range, e.g. 254-2 which expands to 254,255,0,1,2
		if se[0] > se[1] {
			iterRanges[i][0], iterRanges[i][1] = int(se[0]), int(se[0])+int(se[1])
		} else {
			iterRanges[i][0], iterRanges[i][1] = int(se[0]), int(se[1])
		}
	}

	for i := iterRanges[0][0]; i <= iterRanges[0][1]; i++ {
		for j := iterRanges[1][0]; i <= iterRanges[1][1]; i++ {
			for k := iterRanges[2][0]; i <= iterRanges[2][1]; i++ {
				for l := iterRanges[3][0]; i <= iterRanges[3][1]; i++ {
					out = append(out, net.IPv4(byte(i), byte(j), byte(k), byte(l)))
				}
			}
		}
	}

	return
}

func parse(source Input) ([4][2]byte, error) {
	var sink outIPv4
	checker := func(str string) (value byte, err error) {
		i, err := strconv.Atoi(str)
		if i > 255 {
			err = fmt.Errorf("%d overflows byte", value)
		}
		value = byte(i)
		return
	}

	//DOT digraph dashed{
	//States declaration
	var (
		//Terminal states
		//DOT node [shape=doublecircle];
		//DOT startByte;
		//DOT endByte;
		//DOT dash;
		startByte State
		endByte   State
		dash      State
		//Nonterminal states
		//DOT node [shape=circle];
		//DOT begin;
		//DOT dot;
		begin State
		dot   State
	)

	var err error
	//Transitions implementations
	//The start
	begin = func(source Input, sink chan byte) (State, error) {
		//Let's see what's next
		switch c := rune(source.input[source.current]); {
		case unicode.IsNumber(c):
			//DOT begin -> startByte[label="\d"];
			return startByte, nil
		case c == '-':
			//DOT begin -> dash[label="- \n → 0"];
			//No starting value, default to 0
			sink <- 0
			return dash, nil
		default:
			return nil, fmt.Errorf("Unexpected %s at index 0 of ", string(c), source.input)
		}
	}

	//The first or the only byte of a range
	startByte = func(source Input, sink chan byte) (State, error) {
		//Move cursor
		source.current++
		//Input is finished, let's hope everithing went well and exit gently using the
		//read byte as begin and start
		if source.current >= len(source.input) {
			value, err := checker(source.input[source.token_begin:source.current])
			if err != nil {
				return nil, err
			}
			sink <- value
			sink <- value
			return nil, io.EOF
		}
		switch c := rune(source.input[source.current]); {
		case unicode.IsNumber(c):
			//DOT startByte -> startByte[label="\d"];
			//This was a digit, let's see what's next
			//100.0-10.0-.1
			//^ ↑
			return startByte, nil
		case c == '-':
			//DOT startByte -> dash[label="-\n → token"];
			//We have read all the starting byte specifier, let's output it and
			//move to the dash state
			//100.0-10.0-.1
			//    ^↑
			value, err := checker(source.input[source.token_begin:source.current])
			sink <- value
			return dash, err
		case c == '.':
			//DOT startByte -> dot[label=".\n → token, token"];
			//No range specified, let's write the number read until now as both
			//start and end of the range
			//100.0-10.0-.1
			//^  ↑
			value, err := checker(source.input[source.token_begin:source.current])
			sink <- value
			sink <- value
			return dot, err
		default:
			return nil, fmt.Errorf("Unexpected %s at index %d", string(c), source.current)
		}
	}
	//The byte specified after a dash
	endByte = func(source Input, sink chan byte) (State, error) {
		//Move cursor
		source.current++
		//Input is finished, let's hope everithing went well and exit gently
		if source.current >= len(source.input) {
			value, err := checker(source.input[source.token_begin:source.current])
			if err != nil {
				return nil, err
			}
			sink <- value
			close(sink)
			return nil, nil
		}
		switch c := rune(source.input[source.current]); {
		case unicode.IsNumber(c):
			//DOT endByte -> endByte[label="\d"];
			//This was a digit, let's see what's next
			//100.0-10.0-.1
			//      ^↑
			return endByte, nil
		case c == '.':
			//DOT endByte -> dot[label=".\n → token"];
			//We matched a dot, let's emit the read end of the range and move to the dot phase
			//100.0-10.0-.1
			//      ^ ↑
			value, err := checker(source.input[source.token_begin:source.current])
			sink <- value
			return dot, err
		default:
			return nil, fmt.Errorf("Unexpected %s at index %d", string(c), source.current)
		}
	}
	//The dash
	dash = func(source Input, sink chan byte) (State, error) {
		//Move cursor
		source.current++
		//Input is finished, let's hope everithing went well and exit gently, default to 255 as
		//range end
		if source.current >= len(source.input) {
			sink <- 255
			close(sink)
			return nil, nil
		}
		switch c := rune(source.input[source.current]); {
		case unicode.IsNumber(c):
			source.token_begin = source.current
			//DOT dash -> endByte[label="\d\nmove cursor"];
			//An end byte specification is encountered.
			//Let's move the token start here.
			//100.0-10.0-.1
			//      ^
			//      ↑
			return endByte, nil
		case c == '.':
			//DOT dash -> dot[label=".\n → 255"];
			//No end value, default to 255
			//100.0-10.0-.1
			//         ^ ↑
			sink <- 255
			return dot, nil
		default:
			return nil, fmt.Errorf("Unexpected %s at index %d", string(c), source.current)
		}
	}
	//The dot
	dot = func(source Input, sink chan byte) (State, error) {
		//Move cursor
		source.current++
		//Dot can't be the last character
		if source.current >= len(source.input) {
			return nil, fmt.Errorf("Unexpected end of IP: %s", source.input)
		}
		switch c := rune(source.input[source.current]); {
		case unicode.IsNumber(c):
			//DOT dot-> startByte[label="\d\nmove cursor"];
			source.token_begin = source.current
			//A first byte specification is encountered.
			//Let's move the token start here.
			//100.0-10.0-.1
			//            ^
			//            ↑
			return startByte, nil
		case c == '-':
			//DOT dot -> dash[label="-\n → 0"];
			//No starting value, default to 0
			//100.0-10.0-.-1
			//         ^
			//            ↑
			sink <- 0
			return dash, nil
		default:
			return nil, fmt.Errorf("Unexpected %s at index %d", string(c), source.current)
		}
	}
	//DOT }
	state := begin
	//Lex can at most emit 2 tokens per call
	out := make(chan byte, 2)
	//Let's now parse the IP
parse:
	for err != nil {
		//Parser
		select {
		case token, ok := <-out:
			if !ok {
				//Someone closed the output channel, exit the parser
				//Set error if premature end
				if sink.cursor < 8 {
					return sink.out, fmt.Errorf("Unexpected end of IP: %s", source.input)
				}
				err = nil
				break parse
			}
			//TODO if start > end the range should overflow: 254-3 should become 254,255,0,1,2,3
			//if sink.cursor%2 == 1 && sink.out[sink.cursor-1] > sink.out[sink.cursor] {
			//sink.out[sink.cursor], sink.out[sink.cursor-1] = sink.out[sink.cursor-1], sink.out[sink.cursor]
			//}
			err := sink.Write(token)
			if err != nil {
				break parse
			}
			//There might be something else to parse, try to parse it before lex is called
			continue
		default:
		}
		//TODO Parse here instead of checker
		state, err = state(source, out)
	}
	//drain the out channel in case we didn't parse everything before aborting
	for _ = range out {
	}
	return sink.out, nil
}