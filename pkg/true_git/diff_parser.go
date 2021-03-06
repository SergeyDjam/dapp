package true_git

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func makeDiffParser(out io.Writer, pathFilter PathFilter) *diffParser {
	return &diffParser{
		PathFilter:  pathFilter,
		Out:         out,
		OutLines:    0,
		Paths:       make([]string, 0),
		BinaryPaths: make([]string, 0),
		state:       unrecognized,
		lineBuf:     make([]byte, 0, 4096),
	}
}

type parserState string

const (
	unrecognized       parserState = "unrecognized"
	diffBegin          parserState = "diffBegin"
	diffBody           parserState = "diffBody"
	newFileDiff        parserState = "newFileDiff"
	deleteFileDiff     parserState = "deleteFileDiff"
	modifyFileDiff     parserState = "modifyFileDiff"
	modifyFileModeDiff parserState = "modifyFileModeDiff"
	ignoreDiff         parserState = "ignoreDiff"
)

type diffParser struct {
	PathFilter PathFilter

	Out                 io.Writer
	OutLines            uint
	UnrecognizedCapture bytes.Buffer

	Paths         []string
	BinaryPaths   []string
	LastSeenPaths []string

	state   parserState
	lineBuf []byte
}

func appendUnique(list []string, value string) []string {
	for _, oldValue := range list {
		if value == oldValue {
			return list
		}
	}
	return append(list, value)
}

func (p *diffParser) HandleStdout(data []byte) error {
	for _, b := range data {
		if b == '\n' {
			line := string(p.lineBuf)
			p.lineBuf = p.lineBuf[:0]

			err := p.handleDiffLine(line)
			if err != nil {
				return fmt.Errorf("error parsing diff line: %s", err)
			}

			continue
		}
		p.lineBuf = append(p.lineBuf, b)
	}

	return nil
}

func (p *diffParser) HandleStderr(data []byte) error {
	_, err := p.UnrecognizedCapture.Write(data)
	return err
}

func (p *diffParser) writeOutLine(line string) error {
	p.OutLines++

	_, err := p.Out.Write([]byte(line + "\n"))

	return err
}

func (p *diffParser) writeUnrecognizedLine(line string) error {
	_, err := p.UnrecognizedCapture.Write([]byte(line + "\n"))
	return err
}

func (p *diffParser) handleDiffLine(line string) error {
	switch p.state {
	case unrecognized:
		if strings.HasPrefix(line, "diff --git ") {
			return p.handleDiffBegin(line)
		}
		if strings.HasPrefix(line, "Submodule ") {
			return p.handleSubmoduleLine(line)
		}
		return p.writeUnrecognizedLine(line)

	case ignoreDiff:
		if strings.HasPrefix(line, "diff --git ") {
			return p.handleDiffBegin(line)
		}
		if strings.HasPrefix(line, "Submodule ") {
			return p.handleSubmoduleLine(line)
		}
		return nil

	case diffBegin:
		if strings.HasPrefix(line, "deleted file mode ") {
			return p.handleDeleteFileDiff(line)
		}
		if strings.HasPrefix(line, "new file mode ") {
			return p.handleNewFileDiff(line)
		}
		if strings.HasPrefix(line, "old mode ") {
			return p.handleModifyFileModeDiff(line)
		}
		if strings.HasPrefix(line, "index ") {
			return p.handleModifyFileDiff(line)
		}
		return fmt.Errorf("unexpected diff line in state `%s`: %#v", p.state, line)

	case modifyFileDiff:
		if strings.HasPrefix(line, "--- ") {
			return p.handleModifyFilePathA(line)
		}
		if strings.HasPrefix(line, "+++ ") {
			return p.handleModifyFilePathB(line)
		}
		if strings.HasPrefix(line, "GIT binary patch") {
			return p.handleBinaryBeginHeader(line)
		}
		if strings.HasPrefix(line, "Binary files") {
			return p.handleShortBinaryHeader(line)
		}
		return p.writeOutLine(line)

	case modifyFileModeDiff:
		if strings.HasPrefix(line, "new mode ") {
			p.state = unrecognized
			return p.writeOutLine(line)
		}
		return fmt.Errorf("unexpected diff line in state `%s`: %#v", p.state, line)

	case newFileDiff:
		if strings.HasPrefix(line, "+++ ") {
			return p.handleNewFilePath(line)
		}
		if strings.HasPrefix(line, "GIT binary patch") {
			return p.handleBinaryBeginHeader(line)
		}
		if strings.HasPrefix(line, "Binary files") {
			return p.handleShortBinaryHeader(line)
		}
		if strings.HasPrefix(line, "diff --git ") {
			return p.handleDiffBegin(line)
		}
		if strings.HasPrefix(line, "Submodule ") {
			return p.handleSubmoduleLine(line)
		}
		return p.writeOutLine(line)

	case deleteFileDiff:
		if strings.HasPrefix(line, "--- ") {
			return p.handleDeleteFilePath(line)
		}
		if strings.HasPrefix(line, "GIT binary patch") {
			return p.handleBinaryBeginHeader(line)
		}
		if strings.HasPrefix(line, "Binary files") {
			return p.handleShortBinaryHeader(line)
		}
		if strings.HasPrefix(line, "diff --git ") {
			return p.handleDiffBegin(line)
		}
		if strings.HasPrefix(line, "Submodule ") {
			return p.handleSubmoduleLine(line)
		}
		return p.writeOutLine(line)

	case diffBody:
		if strings.HasPrefix(line, "diff --git ") {
			return p.handleDiffBegin(line)
		}
		if strings.HasPrefix(line, "Submodule ") {
			return p.handleSubmoduleLine(line)
		}
		return p.writeOutLine(line)
	}

	return nil
}

func (p *diffParser) handleDiffBegin(line string) error {
	lineParts := strings.Split(line, " ")

	a, b := lineParts[2], lineParts[3]

	trimmedPaths := make(map[string]string)

	p.LastSeenPaths = nil

	for _, data := range []struct{ PathWithPrefix, Prefix string }{{a, "a/"}, {b, "b/"}} {
		if strings.HasPrefix(data.PathWithPrefix, "\"") && strings.HasSuffix(data.PathWithPrefix, "\"") {
			pathWithPrefix, err := strconv.Unquote(data.PathWithPrefix)
			if err != nil {
				return fmt.Errorf("unable to unqoute diff path %#v: %s", data.PathWithPrefix, err)
			}

			path := strings.TrimPrefix(pathWithPrefix, data.Prefix)
			if !p.PathFilter.IsFilePathValid(path) {
				p.state = ignoreDiff
				return nil
			}

			newPath := p.PathFilter.TrimFileBasePath(path)
			p.Paths = appendUnique(p.Paths, newPath)
			p.LastSeenPaths = appendUnique(p.LastSeenPaths, newPath)

			newPathWithPrefix := data.Prefix + newPath
			trimmedPaths[data.PathWithPrefix] = strconv.Quote(newPathWithPrefix)
		} else {
			path := strings.TrimPrefix(data.PathWithPrefix, data.Prefix)
			if !p.PathFilter.IsFilePathValid(path) {
				p.state = ignoreDiff
				return nil
			}

			newPath := p.PathFilter.TrimFileBasePath(path)
			p.Paths = appendUnique(p.Paths, newPath)
			p.LastSeenPaths = appendUnique(p.LastSeenPaths, newPath)

			trimmedPaths[data.PathWithPrefix] = data.Prefix + newPath
		}
	}

	newLine := fmt.Sprintf("diff --git %s %s", trimmedPaths[a], trimmedPaths[b])

	p.state = diffBegin

	return p.writeOutLine(newLine)
}

func (p *diffParser) handleDeleteFileDiff(line string) error {
	p.state = deleteFileDiff
	return p.writeOutLine(line)
}

func (p *diffParser) handleNewFileDiff(line string) error {
	p.state = newFileDiff
	return p.writeOutLine(line)
}

func (p *diffParser) handleModifyFileDiff(line string) error {
	p.state = modifyFileDiff
	return p.writeOutLine(line)
}

func (p *diffParser) handleModifyFileModeDiff(line string) error {
	p.state = modifyFileModeDiff
	return p.writeOutLine(line)
}

func (p *diffParser) handleModifyFilePathA(line string) error {
	path := strings.TrimPrefix(line, "--- a/")
	newPath := p.PathFilter.TrimFileBasePath(path)
	newLine := fmt.Sprintf("--- a/%s", newPath)

	return p.writeOutLine(newLine)
}

func (p *diffParser) handleModifyFilePathB(line string) error {
	path := strings.TrimPrefix(line, "+++ b/")
	newPath := p.PathFilter.TrimFileBasePath(path)
	newLine := fmt.Sprintf("+++ b/%s", newPath)

	p.state = diffBody

	return p.writeOutLine(newLine)
}

func (p *diffParser) handleSubmoduleLine(line string) error {
	p.state = unrecognized
	return nil
}

func (p *diffParser) handleNewFilePath(line string) error {
	path := strings.TrimPrefix(line, "+++ b/")
	newPath := p.PathFilter.TrimFileBasePath(path)
	newLine := fmt.Sprintf("+++ b/%s", newPath)

	p.state = diffBody

	return p.writeOutLine(newLine)
}

func (p *diffParser) handleDeleteFilePath(line string) error {
	path := strings.TrimPrefix(line, "--- a/")
	newPath := p.PathFilter.TrimFileBasePath(path)
	newLine := fmt.Sprintf("--- a/%s", newPath)

	p.state = diffBody

	return p.writeOutLine(newLine)
}

func (p *diffParser) handleBinaryBeginHeader(line string) error {
	for _, path := range p.LastSeenPaths {
		p.BinaryPaths = appendUnique(p.BinaryPaths, path)
	}

	p.state = diffBody

	return p.writeOutLine(line)
}

func (p *diffParser) handleShortBinaryHeader(line string) error {
	for _, path := range p.LastSeenPaths {
		p.BinaryPaths = appendUnique(p.BinaryPaths, path)
	}

	p.state = unrecognized

	return p.writeOutLine(line)
}
