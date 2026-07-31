import { useState, useMemo, useEffect } from 'react';

export function useViewSettings() {
    const initialSettings = useMemo(() => {
        try {
            const saved = localStorage.getItem('viewSettings');
            if (saved) return JSON.parse(saved);
        } catch (e) {}
        return null;
    }, []);
    
    const [showLineNumbers, setShowLineNumbers] = useState<boolean>(initialSettings?.showLineNumbers ?? true);
    const [showDate, setShowDate] = useState<boolean>(initialSettings?.showDate ?? true);
    const [showLevel, setShowLevel] = useState<boolean>(initialSettings?.showLevel ?? true);
    const [showMilliseconds, setShowMilliseconds] = useState<boolean>(initialSettings?.showMilliseconds ?? false);
    const [showTimezone, setShowTimezone] = useState<boolean>(initialSettings?.showTimezone ?? false);
    const [compactMode, setCompactMode] = useState<boolean>(initialSettings?.compactMode ?? false);
    
    const [columnWidths, setColumnWidths] = useState<Record<string, number>>(initialSettings?.columnWidths ?? {
        line: 60,
        lineCompact: 100,
        date: 90,
        dateCompact: 190,
        time: 180,
        timeCompact: 380,
        level: 70,
        tag: 100,
        count: 60
    });
    
    useEffect(() => {
        localStorage.setItem('viewSettings', JSON.stringify({
            compactMode,
            showLineNumbers,
            showDate,
            showLevel,
            showMilliseconds,
            showTimezone,
            columnWidths
        }));
    }, [compactMode, showLineNumbers, showDate, showLevel, showMilliseconds, showTimezone, columnWidths]);

    return {
        showLineNumbers, setShowLineNumbers,
        showDate, setShowDate,
        showLevel, setShowLevel,
        showMilliseconds, setShowMilliseconds,
        showTimezone, setShowTimezone,
        compactMode, setCompactMode,
        columnWidths, setColumnWidths
    };
}
