package main

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const alphabet = "abcdefghijklmnopqrstuvwxyz"

func main() {
	fmt.Println("=== Pro Guard Dictionary Generator ===")
	fmt.Println()

	fmt.Println("[1] Recommended settings (980 words, length 4-9)")
	fmt.Println("[2] Manual")
	fmt.Println()

	var count, minLength, maxLength int
	switch readOption("Choose an option: ", 1, 2) {
	case 1:
		count, minLength, maxLength = 980, 4, 9
	case 2:
		count = readInt("How many combinations to generate? ")
		minLength = readInt("Minimum word length? ")
		maxLength = readMinMax("Maximum word length? ", minLength)
	}

	fmt.Println()
	fmt.Printf("Generating %d unique words (length %d-%d)...\n", count, minLength, maxLength)

	generated := make(map[string]struct{}, count)
	sb := strings.Builder{}

	for len(generated) < count {
		length := minLength + rand.Intn(maxLength-minLength+1)
		sb.Reset()
		for i := range length {
			c := alphabet[rand.Intn(len(alphabet))]
			if i > 0 && rand.Float64() < 0.40 {
				sb.WriteByte(c - 32)
			} else {
				sb.WriteByte(c)
			}
		}
		generated[sb.String()] = struct{}{}
	}

	header := []string{
		"# ╔══════════════════════════════════╗",
		"# ║   Pro Guard Keygen  ·  By Mk     ║",
		"# ╚══════════════════════════════════╝",
		fmt.Sprintf("# Words: %d  |  Length: %d-%d", count, minLength, maxLength),
		"",
	}

	words := make([]string, 0, count)
	for w := range generated {
		words = append(words, w)
	}

	var lines []string
	for i := 0; i < len(words); i += 10 {
		end := i + 10
		if end > len(words) {
			end = len(words)
		}
		lines = append(lines, strings.Join(words[i:end], " "))
	}

	outputPath := filepath.Join(filepath.Dir(os.Args[0]), "output.txt")
	f, err := os.Create(outputPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error creating output file: %v\n", err)
		os.Exit(1)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for _, line := range append(header, lines...) {
		fmt.Fprintln(w, line)
	}
	w.Flush()

	fmt.Println()
	fmt.Printf("Done! %d words saved to:\n", count)
	fmt.Println(outputPath)
	fmt.Println()
	fmt.Print("Press any key to exit...")
	fmt.Scanln()
}

var stdin = bufio.NewReader(os.Stdin)

func readOption(prompt string, minVal, maxVal int) int {
	for {
		fmt.Print(prompt)
		input, _ := stdin.ReadString('\n')
		input = strings.TrimSpace(input)
		if v, err := strconv.Atoi(input); err == nil && v >= minVal && v <= maxVal {
			return v
		}
		fmt.Printf("Invalid option. Enter %d or %d.\n", minVal, maxVal)
	}
}

func readInt(prompt string) int {
	for {
		fmt.Print(prompt)
		input, _ := stdin.ReadString('\n')
		input = strings.TrimSpace(input)
		if v, err := strconv.Atoi(input); err == nil && v > 0 {
			return v
		}
		fmt.Println("Invalid value. Enter a positive integer.")
	}
}

func readMinMax(prompt string, minVal int) int {
	for {
		fmt.Print(prompt)
		input, _ := stdin.ReadString('\n')
		input = strings.TrimSpace(input)
		if v, err := strconv.Atoi(input); err == nil && v >= minVal {
			return v
		}
		fmt.Printf("Invalid value. Must be >= %d.\n", minVal)
	}
}
