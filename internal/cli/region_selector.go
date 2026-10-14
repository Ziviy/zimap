package cli

import (
	"bufio"
	"fmt"
	"zimap/internal/geofabrik"
	"io"
	"strconv"
	"strings"
	"unicode"
)

func selectGeofabrikRegion(scanner *bufio.Scanner, output io.Writer, catalog *geofabrik.Catalog) (*geofabrik.Region, error) {
	var current *geofabrik.Region
	for {
		var choices []*geofabrik.Region
		if current == nil {
			fmt.Fprintln(output, "\nВыберите регион:")
			choices = catalog.Roots()
		} else {
			fmt.Fprintf(output, "\n%s\n\n", current.Name)
			if current.PBFURL != "" {
				fmt.Fprintf(output, "0. Скачать %s\n", current.Name)
			}
			choices = catalog.Children(current.ID)
		}

		for index, region := range choices {
			fmt.Fprintf(output, "%d. %s\n", index+1, region.Name)
		}
		if current != nil {
			fmt.Fprintln(output, "b. Назад")
		}
		fmt.Fprintln(output, "q. Выход")
		fmt.Fprint(output, "> ")

		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				return nil, fmt.Errorf("read region selection: %w", err)
			}
			return nil, nil
		}
		answer := strings.TrimSpace(scanner.Text())
		switch strings.ToLower(answer) {
		case "q", "quit", "exit":
			return nil, nil
		case "b", "back":
			if current != nil {
				current = catalog.Parent(current)
				continue
			}
		}

		selection, err := strconv.Atoi(answer)
		if err != nil {
			fmt.Fprintln(output, "Введите номер пункта, b для возврата или q для выхода.")
			continue
		}
		if current != nil && selection == 0 && current.PBFURL != "" {
			return current, nil
		}
		if selection < 1 || selection > len(choices) {
			fmt.Fprintln(output, "Такого пункта нет.")
			continue
		}
		current = choices[selection-1]
	}
}

func confirmRegionDownload(scanner *bufio.Scanner, output io.Writer, region *geofabrik.Region, size int64) (bool, error) {
	fmt.Fprintf(output, "\nРегион: %s\nURL: %s\n", region.Name, region.PBFURL)
	if size >= 0 {
		fmt.Fprintf(output, "Размер: %s\n", formatByteSize(size))
	}
	fmt.Fprint(output, "Скачать и конвертировать? [y/N] ")
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return false, fmt.Errorf("read download confirmation: %w", err)
		}
		return false, nil
	}
	answer := strings.ToLower(strings.TrimSpace(scanner.Text()))
	return answer == "y" || answer == "yes" || answer == "д" || answer == "да", nil
}

func regionFileName(name string) string {
	var result strings.Builder
	separatorPending := false
	for _, character := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			if separatorPending && result.Len() > 0 {
				result.WriteByte('-')
			}
			result.WriteRune(character)
			separatorPending = false
			continue
		}
		separatorPending = result.Len() > 0
	}
	if result.Len() == 0 {
		return "region"
	}
	return result.String()
}

func formatByteSize(size int64) string {
	const unit = int64(1024)
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	divisor, exponent := unit, 0
	for value := size / unit; value >= unit && exponent < 4; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(size)/float64(divisor), "KMGTPE"[exponent])
}
