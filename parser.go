package main

import (
	"fmt"
	"regexp"
	"strings"
	"sync/atomic"
	"time"
)

var idCounter int64

var (
	shadowRegex    = regexp.MustCompile(`^(\d{4}/\d{2}/\d{2} \d{2}:\d{2}:\d{2})\s+\[(.+?)\]\s+(.*?)\s+hash=([a-zA-Z0-9]+)\s+:(\d+)\s+→\s+(.*?)\s+→\s+(.*)$`)
	bracketRegex   = regexp.MustCompile(`^\[(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2})\]\s+(.*)$`)
	pythonRegex    = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2},\d{3})\s+-\s+(.*)$`)
	kvTimeRegex    = regexp.MustCompile(`time="?([^"\s]+)"?`)
	kvTagRegex     = regexp.MustCompile(`tag="?([^"\s]+)"?`)
	kvLevelRegex   = regexp.MustCompile(`level="?([^"\s]+)"?`)
	timeSplitRegex = regexp.MustCompile(`^(\d{4}[-/]\d{2}[-/]\d{2})\s+(\d{2}:\d{2}:\d{2})(?:[,.](\d+))?\s*(Z|[+-]\d{2}:?\d{2})?`)
)

func parseTimeComponents(rawTime string) (date, timePart, ms, tz string) {
	// Normalize T delimiter from ISO format to space for regex matching
	t := strings.Replace(rawTime, "T", " ", 1)
	
	matches := timeSplitRegex.FindStringSubmatch(t)
	if len(matches) > 0 {
		date = matches[1]
		timePart = matches[2]
		ms = matches[3]
		tz = matches[4]
	} else {
		// Fallback if it doesn't strictly match the expected YYYY-MM-DD HH:MM:SS format
		timePart = rawTime
	}
	return
}

func (a *App) parseLine(text string, sessionID int64) {
	a.mu.Lock()
	if sessionID != a.sessionID {
		a.mu.Unlock()
		return
	}
	isShadow := a.isShadow
	a.mu.Unlock()
	
	entry := a.parseSingleLine(text, isShadow)

	a.mu.Lock()
	a.currentLastLineNum++
	entry.LineNum = a.currentLastLineNum
	a.logEntries = append(a.logEntries, entry)
	
	// Keep maximum 50000 entries in memory for live tail
	if len(a.logEntries) > 50000 {
		a.logEntries = a.logEntries[1:]
		if a.lastSentIdx > 0 {
			a.lastSentIdx--
		}
	}
	a.mu.Unlock()
}

func (a *App) parseSingleLine(text string, isShadow bool) LogEntry {
	text = strings.TrimRight(text, "\r\n")
	var entry LogEntry
	id := atomic.AddInt64(&idCounter, 1)
	entry.Id = fmt.Sprintf("%d-%d", time.Now().UnixNano(), id)
	entry.Raw = text
	entry.Count = 1
	
	if isShadow {
		matches := shadowRegex.FindStringSubmatch(text)
		if len(matches) >= 8 {
			d, t, ms, tz := parseTimeComponents(matches[1])
			entry.Date = d
			entry.Time = t
			entry.Ms = ms
			entry.Tz = tz
			entry.Tag = matches[2]
			entry.Message = fmt.Sprintf("%s hash=%s :%s → %s → %s", matches[3], matches[4], matches[5], matches[6], matches[7])
		} else {
			entry.Message = text
		}
	} else {
		msg := text

		// Step 1: Extract Time
		rawTime := ""
		if matches := bracketRegex.FindStringSubmatch(msg); len(matches) >= 3 {
			rawTime = matches[1]
			msg = matches[2]
		} else if matches := pythonRegex.FindStringSubmatch(msg); len(matches) >= 3 {
			rawTime = matches[1]
			msg = matches[2]
		} else if matches := kvTimeRegex.FindStringSubmatch(msg); len(matches) > 1 {
			rawTime = matches[1]
			msg = kvTimeRegex.ReplaceAllString(msg, "")
		} else {
			if loc := timeSplitRegex.FindStringIndex(msg); loc != nil {
				rawTime = strings.TrimSpace(msg[loc[0]:loc[1]])
				msg = strings.TrimSpace(msg[loc[1]:])
			}
		}
		
		if rawTime != "" {
			d, t, ms, tz := parseTimeComponents(rawTime)
			entry.Date = d
			entry.Time = t
			entry.Ms = ms
			entry.Tz = tz
		}

		// Step 2: Extract explicit Key-Value metadata
		if m := kvLevelRegex.FindStringSubmatch(msg); len(m) > 1 {
			entry.Level = m[1]
			msg = kvLevelRegex.ReplaceAllString(msg, "")
		}
		if m := kvTagRegex.FindStringSubmatch(msg); len(m) > 1 {
			entry.Tag = m[1]
			msg = kvTagRegex.ReplaceAllString(msg, "")
		}

		msg = strings.TrimSpace(msg)

		// Step 3 & 4: Extract preset levels and remaining leading brackets
		for i := 0; i < 2; i++ {
			if strings.HasPrefix(msg, "[") {
				endIdx := strings.Index(msg, "]")
				if endIdx > 0 {
					content := msg[1:endIdx]
					
					isLevel := false
					lowerContent := strings.ToLower(content)
					if lowerContent == "info" || lowerContent == "error" || lowerContent == "warn" || lowerContent == "warning" || lowerContent == "debug" || lowerContent == "fatal" || lowerContent == "trace" {
						isLevel = true
					}
					
					if isLevel && entry.Level == "" {
						entry.Level = content
					} else if entry.Tag == "" {
						entry.Tag = content
					} else {
						// Both Level and Tag are filled, stop stripping brackets
						break
					}
					
					// Consume the bracket from the message
					msg = strings.TrimSpace(msg[endIdx+1:])
				} else {
					break // No matching end bracket found
				}
			} else {
				break // Does not start with a bracket
			}
		}

		// Step 4.5: Check for unbracketed level prefixes like "Warning:" or "ERROR:"
		if entry.Level == "" {
			spaceIdx := strings.Index(msg, " ")
			if spaceIdx > 0 {
				firstWord := msg[:spaceIdx]
				cleanWord := strings.TrimRight(firstWord, ":-")
				lowerWord := strings.ToLower(cleanWord)
				if lowerWord == "info" || lowerWord == "error" || lowerWord == "warn" || lowerWord == "warning" || lowerWord == "debug" || lowerWord == "fatal" || lowerWord == "trace" {
					entry.Level = cleanWord
					msg = strings.TrimSpace(msg[spaceIdx:])
				}
			}
		}

		// Step 5: Finalize Message
		entry.Message = strings.TrimSpace(msg)
	}
	
	entry.EndDate = entry.Date
	entry.EndTime = entry.Time
	entry.EndMs = entry.Ms
	entry.EndTz = entry.Tz
	return entry
}
