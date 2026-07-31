import { useState, useEffect, useRef, useMemo } from 'react';
import { models } from '../../wailsjs/go/models';
import { StartTailing, StopTailing, LoadPreviousChunk } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export function useLogs(compactMode: boolean) {
    const [activeFile, setActiveFile] = useState<models.FileInfo | null>(null);
    const [logs, setLogs] = useState<models.LogEntry[]>([]);
    const [searchTerm, setSearchTerm] = useState<string>('');
    const [autoScroll, setAutoScroll] = useState<boolean>(true);
    const [firstItemIndex, setFirstItemIndex] = useState<number>(1000000);
    
    const isLoadingMore = useRef(false);
    const activeFilePathRef = useRef<string | null>(null);

    const handleSelectFile = async (file: models.FileInfo) => {
        activeFilePathRef.current = file.path; // Instantly abort any active loops
        
        setActiveFile(file);
        setLogs([]); // Clear while loading
        setAutoScroll(true); // Re-enable auto-scroll on new file
        setFirstItemIndex(1000000); // Reset index for new file
        
        await StartTailing(file.path);
        
        const currentPath = file.path;
        const initial = await LoadPreviousChunk();
        
        if (activeFilePathRef.current !== currentPath) return; // User clicked another file while waiting
        
        setLogs(initial || []);
    };

    const stopTailingIfActive = async (path: string) => {
        if (activeFilePathRef.current?.startsWith(path)) {
            await StopTailing();
            activeFilePathRef.current = null;
            setActiveFile(null);
            setLogs([]);
        }
    };

    const loadMoreHistory = async () => {
        if (isLoadingMore.current) return;
        
        isLoadingMore.current = true;
        try {
            const chunk = await LoadPreviousChunk();
            if (chunk && chunk.length > 0) {
                setLogs(prev => [...chunk, ...prev]);
                setFirstItemIndex(prev => prev - chunk.length);
            }
        } finally {
            isLoadingMore.current = false;
        }
    };

    useEffect(() => {
        // Listen for live updates (Delta payloads)
        EventsOn('log_update', (update: any) => {
            setLogs(prev => {
                let next = update.clearLogs ? [] : [...prev];
                if (update.lastEntryUpdate && next.length > 0) {
                    next[next.length - 1] = update.lastEntryUpdate;
                }
                if (update.newEntries && update.newEntries.length > 0) {
                    next = [...next, ...update.newEntries];
                }
                return next;
            });
        });
    }, []);

    const displayLogs = useMemo(() => {
        let result = logs;
        if (searchTerm) {
            const lowerSearch = searchTerm.toLowerCase();
            result = result.filter(l => 
                (l.message || '').toLowerCase().includes(lowerSearch) ||
                (l.tag || '').toLowerCase().includes(lowerSearch) ||
                (l.raw || '').toLowerCase().includes(lowerSearch)
            );
        }

        if (!compactMode) {
            return result; // Skip expensive mapping if not compacting
        }

        const compacted: any[] = [];
        let current: any = null;

        for (const log of result) {
            if (!current) {
                current = { 
                    ...log, 
                    count: 1, 
                    startLine: log.lineNum, 
                    endLine: log.lineNum,
                    startDate: log.date,
                    startTime: log.time,
                    startMs: log.ms,
                    startTz: log.tz,
                    endDate: log.date,
                    endTime: log.time, 
                    endMs: log.ms,
                    endTz: log.tz
                };
                compacted.push(current);
            } else {
                if (current.tag === log.tag && current.message === log.message) {
                    current.count++;
                    if (log.time) {
                        current.endDate = log.date;
                        current.endTime = log.time;
                        current.endMs = log.ms;
                        current.endTz = log.tz;
                    }
                    current.endLine = log.lineNum;
                } else {
                    current = { 
                        ...log, 
                        count: 1, 
                        startLine: log.lineNum, 
                        endLine: log.lineNum,
                        startDate: log.date,
                        startTime: log.time,
                        startMs: log.ms,
                        startTz: log.tz,
                        endDate: log.date,
                        endTime: log.time, 
                        endMs: log.ms,
                        endTz: log.tz
                    };
                    compacted.push(current);
                }
            }
        }
        return compacted;
    }, [logs, searchTerm, compactMode]);

    return {
        activeFile,
        logs,
        displayLogs,
        searchTerm, setSearchTerm,
        autoScroll, setAutoScroll,
        firstItemIndex,
        handleSelectFile,
        loadMoreHistory,
        stopTailingIfActive
    };
}
