package parsers

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"unicode"
)

type State func(*Input, chan string) (State, error)

type Input struct {
	//TODO use []rune instead, or use proper rune by rune scanning
	buf        string
	tokenStart int
	cur        int
}

//Returns the current rune. Does not check for read out of boundaries
func (i *Input) GetRune() rune {
	//TODO use appropiate cursor for runes
	return rune(i.buf[i.cur])
}
func (i *Input) Step() error {
	i.cur++
	if i.cur >= len(i.buf) {
		return io.EOF
	}
	return nil
}
func (i *Input) GetToken() string {
	//TODO user proper slicing
	return i.buf[i.tokenStart:i.cur]
}
func (i *Input) AdvanceTokenBegin() {
	i.tokenStart = i.cur
}
func (i *Input) Cursor() int {
	return i.cur
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
		log.Println(err)
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
		for j := iterRanges[1][0]; j <= iterRanges[1][1]; j++ {
			for k := iterRanges[2][0]; k <= iterRanges[2][1]; k++ {
				for l := iterRanges[3][0]; l <= iterRanges[3][1]; l++ {
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
		if err != nil {
			return
		}
		if i > 255 {
			err = fmt.Errorf("%d overflows byte", i)
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
	begin = func(source *Input, sink chan string) (State, error) {
		//Let's see what's next
		//TODO handle null input
		log.Printf("Begin (%s)\n", string(source.GetRune()))
		switch c := source.GetRune(); {
		case unicode.IsNumber(c):
			//DOT begin -> startByte[label="\d"];
			return startByte, nil
		case c == '-':
			//DOT begin -> dash[label="- \n → 0"];
			//No starting value, default to 0
			sink <- "0"
			return dash, nil
		default:
			return nil, fmt.Errorf("unexpected %s at index %d", string(c), source.Cursor())
		}
	}

	//The first or the only byte of a range
	startByte = func(source *Input, sink chan string) (State, error) {
		//Move cursor
		err := source.Step()
		//Input is finished, let's hope everything went well and exit gently using the
		//read byte as begin and start
		if err == io.EOF {
			sink <- source.GetToken()
			sink <- source.GetToken()
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		log.Printf("Start Byte (%c)\n", source.GetRune())
		switch c := source.GetRune(); {
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
			sink <- source.GetToken()
			return dash, nil
		case c == '.':
			//DOT startByte -> dot[label=".\n → token, token"];
			//No range specified, let's write the number read until now as both
			//start and end of the range
			//100.0-10.0-.1
			//^  ↑
			sink <- source.GetToken()
			sink <- source.GetToken()
			return dot, nil
		default:
			return nil, fmt.Errorf("unexpected %s at index %d", string(c), source.Cursor())
		}
	}
	//The byte specified after a dash
	endByte = func(source *Input, sink chan string) (State, error) {
		log.Println("End Byte")
		//Move cursor
		err := source.Step()
		//Input is finished, let's hope everything went well and exit gently
		if err == io.EOF {
			sink <- source.GetToken()
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		log.Printf("End Byte (%c)\n", source.GetRune())
		switch c := source.GetRune(); {
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
			sink <- source.GetToken()
			return dot, nil
		default:
			return nil, fmt.Errorf("unexpected %s at index %d", string(c), source.Cursor())
		}
	}
	//The dash
	dash = func(source *Input, sink chan string) (State, error) {
		//Move cursor
		err := source.Step()
		//Input is finished, let's hope everything went well and exit gently, default to 255 as
		//range end
		if err == io.EOF {
			sink <- "255"
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		log.Printf("Dash (%c)\n", source.GetRune())
		switch c := source.GetRune(); {
		case unicode.IsNumber(c):
			source.AdvanceTokenBegin()
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
			sink <- "255"
			return dot, nil
		default:
			return nil, fmt.Errorf("unexpected %s at index %d", string(c), source.Cursor())
		}
	}
	//The dot
	dot = func(source *Input, sink chan string) (State, error) {
		//Move cursor
		err := source.Step()
		//Dot can't be the last character
		if err == io.EOF {
			return nil, fmt.Errorf("unexpected end of IP")
		}
		if err != nil {
			return nil, err
		}
		log.Printf("Dot (%c)\n", source.GetRune())
		switch c := source.GetRune(); {
		case unicode.IsNumber(c):
			//DOT dot-> startByte[label="\d\nmove cursor"];
			source.AdvanceTokenBegin()
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
			sink <- "0"
			return dash, nil
		default:
			return nil, fmt.Errorf("unexpected %s at index %d", string(c), source.Cursor())
		}
	}
	//DOT }
	state := begin
	//Lex can at most emit 2 tokens per call
	out := make(chan string, 2)
	//Let's now parse the IP
	log.Println("Starting parser")
parse:
	for {
		//Parser
		select {
		case token, ok := <-out:
			if !ok {
				log.Println("Channel closed")
				//Someone closed the output channel, exit the parser
				//Set error if premature end
				if sink.cursor < 8 {
					err = fmt.Errorf("unexpected end of IP")
					break parse
				}
				break parse
			}
			log.Printf("Parsing lexed token '%s'\n", token)
			var value byte
			value, err = checker(token)
			if err != nil {
				break parse
			}
			err = sink.Write(value)
			if err != nil {
				break parse
			}
			//There might be something else to parse, try to parse it before lex is called
			continue
		default:
		}
		log.Println("Lexing")
		state, err = state(&source, out)
		if state == nil || err != nil {
			//No state was returned, close the sink, shut down the parser
			close(out)
		}
	}
	return sink.out, err
}
