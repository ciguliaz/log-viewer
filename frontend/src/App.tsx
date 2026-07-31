import { useState, useEffect, useMemo, useRef } from 'react';
import './App.css';
import { EventsOn, WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit } from '../wailsjs/runtime/runtime';
import { SelectFolder, ListFiles, StartTailing, StopTailing, LoadPreviousChunk, ProcessDrop, CheckForUpdate, ApplyUpdate } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { TableVirtuoso, TableVirtuosoHandle } from 'react-virtuoso';

const MessageCell = ({ text }: { text: string }) => {
    const [isExpanded, setIsExpanded] = useState(false);
    
    if (!text) return null;

    const MAX_LINES = 7;
    const MAX_CHARS = 1000;
    const lines = text.split('\n');
    
    if ((lines.length <= MAX_LINES && text.length <= MAX_CHARS) || isExpanded) {
        return (
            <div 
                onClick={() => isExpanded && setIsExpanded(false)} 
                style={{ cursor: isExpanded ? 'pointer' : 'text' }}
            >
                {text}
                {isExpanded && <div style={{marginTop: '5px', color: 'var(--accent)', fontSize: '11px', userSelect: 'none', fontWeight: 'bold'}}>Show less...</div>}
            </div>
        );
    }
    
    let topText = "";
    let bottomText = "";
    let hiddenCount = 0;

    if (lines.length > MAX_LINES) {
        topText = lines.slice(0, 3).join('\n');
        bottomText = lines.slice(-2).join('\n');
        hiddenCount = lines.length - 5;
    } else {
        topText = text.substring(0, 300);
        bottomText = text.substring(text.length - 100);
    }
    
    return (
        <div onClick={() => setIsExpanded(true)} style={{ cursor: 'pointer' }}>
            <div>{topText}</div>
            <div style={{ color: 'var(--accent)', margin: '4px 0', fontSize: '11px', userSelect: 'none', fontWeight: 'bold' }}>
                ... {hiddenCount > 0 ? `(${hiddenCount} more lines hidden)` : '(long text hidden)'} Click to expand ...
            </div>
            <div>{bottomText}</div>
        </div>
    );
};

function App() {
    type WorkspaceFolder = {
        path: string;
        name: string;
        files: main.FileInfo[];
        error?: string;
    };
    const [workspaces, setWorkspaces] = useState<WorkspaceFolder[]>([]);
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
    
    const [activeFile, setActiveFile] = useState<main.FileInfo | null>(null);
    const [logs, setLogs] = useState<main.LogEntry[]>([]);
    
    // Update State
    const [updateInfo, setUpdateInfo] = useState<{available: boolean, version: string, releaseNotes: string} | null>(null);
    const [showUpdateModal, setShowUpdateModal] = useState(false);
    const [isUpdating, setIsUpdating] = useState(false);
    
    useEffect(() => {
        const checkUpdate = async () => {
            try {
                // @ts-ignore - in case bindings aren't fully generated yet
                const info = await CheckForUpdate();
                if (info && info.available) {
                    setUpdateInfo(info);
                }
            } catch (e) {
                console.error("Failed to check for updates:", e);
            }
        };
        // Delay update check slightly to not block startup
        setTimeout(checkUpdate, 2000);
    }, []);

    const handleApplyUpdate = async () => {
        setIsUpdating(true);
        try {
            // @ts-ignore
            await ApplyUpdate();
            alert("Update successful! Please restart the application.");
            Quit();
        } catch (e) {
            console.error("Update failed:", e);
            alert("Update failed: " + e);
            setIsUpdating(false);
        }
    };
    
    // Window state for maximize button icon
    const [isMaximized, setIsMaximized] = useState(false);
    
    useEffect(() => {
        const checkMaximize = async () => {
            const max = await WindowIsMaximised();
            setIsMaximized(max);
        };
        
        window.addEventListener('resize', checkMaximize);
        checkMaximize();
        
        return () => window.removeEventListener('resize', checkMaximize);
    }, []);

    // View settings persistence
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

    const [searchTerm, setSearchTerm] = useState<string>('');
    const [autoScroll, setAutoScroll] = useState<boolean>(true);
    const [firstItemIndex, setFirstItemIndex] = useState<number>(1000000);
    const isLoadingMore = useRef(false);
    const activeFilePathRef = useRef<string | null>(null);
    const isInitialMount = useRef(true);
    
    // Load persisted workspaces on mount
    useEffect(() => {
        const loadPersisted = async () => {
            const saved = localStorage.getItem('workspaceFolders');
            if (saved) {
                try {
                    const paths = JSON.parse(saved) as string[];
                    for (const path of paths) {
                        const dropResult = await ProcessDrop(path);
                        if (dropResult) {
                            setWorkspaces(prev => {
                                if (prev.some(w => w.path === dropResult.path)) return prev;
                                return [...prev, {
                                    path: dropResult.path,
                                    name: dropResult.name,
                                    files: dropResult.files || [],
                                    error: dropResult.error || undefined
                                }];
                            });
                            // Automatically expand restored folders
                            setExpandedFolders(prev => new Set(prev).add(dropResult.path));
                        }
                    }
                } catch (e) {
                    console.error("Failed to parse saved workspaces", e);
                }
            }
        };
        loadPersisted();
    }, []);

    // Save workspaces to localStorage when they change
    useEffect(() => {
        if (isInitialMount.current) {
            isInitialMount.current = false;
            return;
        }
        const paths = workspaces.map(w => w.path);
        localStorage.setItem('workspaceFolders', JSON.stringify(paths));
    }, [workspaces]);

    // Periodically poll workspaces to detect if they were renamed/deleted or if new files appeared
    useEffect(() => {
        if (workspaces.length === 0) return;
        const interval = setInterval(async () => {
            const updatedWorkspaces = await Promise.all(workspaces.map(async ws => {
                const dropResult = await ProcessDrop(ws.path);
                if (dropResult) {
                    return {
                        ...ws,
                        files: dropResult.files || [],
                        error: dropResult.error || undefined
                    };
                }
                return ws;
            }));
            
            setWorkspaces(prev => {
                let changed = false;
                if (prev.length !== updatedWorkspaces.length) return updatedWorkspaces;
                
                for (let i = 0; i < prev.length; i++) {
                    if (prev[i].error !== updatedWorkspaces[i].error) {
                        changed = true;
                        break;
                    }
                    if (prev[i].files.length !== updatedWorkspaces[i].files.length) {
                        changed = true;
                        break;
                    }
                    // Deep compare file names to detect renames
                    const prevNames = prev[i].files.map(f => f.name).join('|');
                    const updatedNames = updatedWorkspaces[i].files.map(f => f.name).join('|');
                    if (prevNames !== updatedNames) {
                        changed = true;
                        break;
                    }
                }
                return changed ? updatedWorkspaces : prev;
            });
        }, 3000); // Check every 3 seconds
        
        return () => clearInterval(interval);
    }, [workspaces]);

    const virtuosoRef = useRef<TableVirtuosoHandle>(null);

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

    const handleSelectFolder = async () => {
        const selected = await SelectFolder();
        if (selected) {
            const fileList = await ListFiles(selected);
            const wsName = selected.split('\\').pop() || selected.split('/').pop() || selected;
            setWorkspaces([{
                path: selected,
                name: wsName,
                files: fileList || [],
                error: undefined
            }]);
            setExpandedFolders(new Set([selected]));
        }
    };

    const handleAddFolder = async () => {
        const selected = await SelectFolder();
        if (selected) {
            if (workspaces.some(w => w.path === selected)) return;
            const fileList = await ListFiles(selected);
            const wsName = selected.split('\\').pop() || selected.split('/').pop() || selected;
            setWorkspaces(prev => [...prev, {
                path: selected,
                name: wsName,
                files: fileList || [],
                error: undefined
            }]);
            setExpandedFolders(prev => new Set(prev).add(selected));
        }
    };

    const toggleFolder = (path: string) => {
        setExpandedFolders(prev => {
            const next = new Set(prev);
            if (next.has(path)) next.delete(path);
            else next.add(path);
            return next;
        });
    };

    const handleRemoveFolder = async (e: React.MouseEvent, path: string) => {
        e.stopPropagation();
        setWorkspaces(prev => prev.filter(w => w.path !== path));
        
        // If the active file is inside this folder, we must stop tailing it to release the file lock
        if (activeFilePathRef.current?.startsWith(path)) {
            await StopTailing();
            activeFilePathRef.current = null;
            setActiveFile(null);
            setLogs([]);
        }
    };

    const handleSelectFile = async (file: main.FileInfo) => {
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
        const preventDefault = (e: Event) => {
            e.preventDefault();
            e.stopPropagation();
        };
        window.addEventListener('dragover', preventDefault);
        window.addEventListener('drop', preventDefault);

        // Handle drag and drop via Wails native event (requires --wails-drop-target: drop in CSS)
        EventsOn('wails:file-drop', async (...args: any[]) => {
            // Wails may pass (paths) or (x, y, paths) depending on version
            let paths: string[] = [];
            if (Array.isArray(args[0])) {
                paths = args[0];
            } else if (args.length >= 3 && Array.isArray(args[2])) {
                paths = args[2];
            }
            if (!paths || paths.length === 0) return;
            
            for (const path of paths) {
                const dropResult = await ProcessDrop(path);
                if (dropResult) {
                    setWorkspaces(prev => {
                        if (prev.some(w => w.path === dropResult.path)) return prev;
                        return [...prev, {
                            path: dropResult.path,
                            name: dropResult.name,
                            files: dropResult.files || []
                        }];
                    });
                    setExpandedFolders(prev => new Set(prev).add(dropResult.path));
                }
            }
        });

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

        return () => {
            window.removeEventListener('dragover', preventDefault);
            window.removeEventListener('drop', preventDefault);
        };
    }, []);

    // Auto-scroll logic
    useEffect(() => {
        if (autoScroll && virtuosoRef.current && displayLogs.length > 0) {
            // Use timeout to ensure DOM has updated
            setTimeout(() => {
                virtuosoRef.current?.scrollToIndex({
                    index: displayLogs.length - 1,
                    align: 'end',
                    behavior: 'auto'
                });
            }, 50);
        }
    }, [displayLogs.length, autoScroll]);

    return (
        <div className="app-container">
            <div className="menu-bar">
                <div className="menu-item">
                    File
                    <div className="menu-dropdown">
                        <div className="menu-dropdown-item" onClick={handleSelectFolder}>
                            Open Folder...
                        </div>
                        <div className="menu-dropdown-item" onClick={handleAddFolder}>
                            Add Folder...
                        </div>
                    </div>
                </div>
                <div className="menu-item">
                    View
                    <div className="menu-dropdown">
                        <div className="menu-dropdown-item" onClick={() => setCompactMode(!compactMode)}>
                            <span style={{ width: '20px' }}>{compactMode ? '✓' : ''}</span> Compact Mode
                        </div>
                        <div className="menu-dropdown-item" onClick={() => setShowLineNumbers(!showLineNumbers)}>
                            <span style={{ width: '20px' }}>{showLineNumbers ? '✓' : ''}</span> Line Numbers
                        </div>
                        <div className="menu-dropdown-item" onClick={() => setShowLevel(!showLevel)}>
                            <span style={{ width: '20px' }}>{showLevel ? '✓' : ''}</span> Show Level
                        </div>
                        <div className="menu-dropdown-item menu-submenu-parent">
                            <span style={{ width: '20px' }}></span> Time Format <span style={{ marginLeft: 'auto', fontSize: '10px' }}>▶</span>
                            <div className="menu-submenu">
                                <div className="menu-dropdown-item" onClick={(e) => { e.stopPropagation(); setShowDate(!showDate); }}>
                                    <span style={{ width: '20px' }}>{showDate ? '✓' : ''}</span> Show Date
                                </div>
                                <div className="menu-dropdown-item" onClick={(e) => { e.stopPropagation(); setShowMilliseconds(!showMilliseconds); }}>
                                    <span style={{ width: '20px' }}>{showMilliseconds ? '✓' : ''}</span> Show Milliseconds
                                </div>
                                <div className="menu-dropdown-item" onClick={(e) => { e.stopPropagation(); setShowTimezone(!showTimezone); }}>
                                    <span style={{ width: '20px' }}>{showTimezone ? '✓' : ''}</span> Show Timezone
                                </div>
                            </div>
                        </div>
                    </div>
                </div>
                
                <div className="window-controls">
                    {updateInfo && (
                        <div 
                            className="update-available-text"
                            onClick={() => setShowUpdateModal(true)}
                            style={{ cursor: 'pointer', color: 'var(--accent)', fontSize: '12px', marginRight: '15px', display: 'flex', alignItems: 'center', fontWeight: 'bold' }}
                        >
                            Update Available ({updateInfo.version})
                        </div>
                    )}
                    <div className="window-control" onClick={() => WindowMinimise()}>
                        <svg width="12" height="12" viewBox="0 0 12 12"><rect fill="currentColor" width="10" height="1" x="1" y="6"></rect></svg>
                    </div>
                    <div className="window-control" onClick={() => WindowToggleMaximise()}>
                        {isMaximized ? (
                            <svg width="12" height="12" viewBox="0 0 12 12"><path fill="currentColor" fillRule="evenodd" d="M2.5 4.5v5h5v-5h-5zm-1-1h7v7h-7v-7zm6.5-1h-5v-1h7v7h-1v-5h-1v-1z"></path></svg>
                        ) : (
                            <svg width="12" height="12" viewBox="0 0 12 12"><rect width="9" height="9" x="1.5" y="1.5" fill="none" stroke="currentColor"></rect></svg>
                        )}
                    </div>
                    <div className="window-control close" onClick={() => Quit()}>
                        <svg width="12" height="12" viewBox="0 0 12 12"><polygon fill="currentColor" fillRule="evenodd" points="11 1.576 6.583 6 11 10.424 10.424 11 6 6.583 1.576 11 1 10.424 5.417 6 1 1.576 1.576 1 6 5.417 10.424 1"></polygon></svg>
                    </div>
                </div>
            </div>
            
            <div className="layout">
                <div className="sidebar">
                    <div className="sidebar-header">
                        <span>EXPLORER</span>
                    </div>
                    <div className="file-list">
                        {workspaces.map(ws => (
                            <div key={ws.path} className="workspace-folder">
                                <div 
                                    className="workspace-header" 
                                    onClick={() => toggleFolder(ws.path)}
                                    title={ws.error ? ws.error : ws.path}
                                    style={{ display: 'flex', alignItems: 'center' }}
                                >
                                    <span style={{fontSize: '9px', width: '12px'}}>{expandedFolders.has(ws.path) ? '▼' : '▶'}</span> 
                                    <span style={{flex: 1, whiteSpace: 'nowrap', overflow: 'hidden', textOverflow: 'ellipsis'}}>
                                        {ws.name.toUpperCase()}
                                    </span>
                                    {ws.error && <span style={{ color: '#ff4d4f', marginLeft: '5px', fontSize: '11px' }} title={ws.error}>⚠️</span>}
                                    <span 
                                        className="remove-folder-btn"
                                        onClick={(e) => handleRemoveFolder(e, ws.path)}
                                        title="Remove folder"
                                        style={{ marginLeft: '5px', padding: '0 4px', cursor: 'pointer', fontSize: '12px', color: 'var(--text-dim)' }}
                                    >
                                        ✕
                                    </span>
                                </div>
                                {expandedFolders.has(ws.path) && (
                                    <div className="workspace-files">
                                        {ws.files.map(f => (
                                            <div 
                                                key={f.path} 
                                                className={`file-item ${activeFile?.path === f.path ? 'active' : ''}`}
                                                onClick={() => handleSelectFile(f)}
                                            >
                                                📄 {f.name}
                                            </div>
                                        ))}
                                        {ws.files.length === 0 && (
                                            <div style={{padding: '5px 10px', color: 'var(--text-dim)', fontSize: '12px'}}>No .log files</div>
                                        )}
                                    </div>
                                )}
                            </div>
                        ))}
                    </div>
            </div>
            <div className="main-view">
                {activeFile ? (
                    <>
                        <div className="tabs">
                            <div className="tab">{activeFile.name}</div>
                            <div className="search-container">
                                <input 
                                    type="text" 
                                    placeholder="Search logs..." 
                                    value={searchTerm}
                                    onChange={(e) => setSearchTerm(e.target.value)}
                                    className="search-input"
                                />
                                {searchTerm && <span className="search-count">{displayLogs.length} results</span>}
                            </div>
                        </div>
                        <div className="log-container">
                            <TableVirtuoso
                                ref={virtuosoRef}
                                className="virtuoso-table"
                                data={displayLogs}
                                overscan={500}
                                firstItemIndex={firstItemIndex}
                                initialTopMostItemIndex={displayLogs.length > 0 ? displayLogs.length - 1 : 0}
                                startReached={loadMoreHistory}
                                atBottomStateChange={(atBottom) => {
                                    setAutoScroll(atBottom);
                                }}
                                fixedHeaderContent={() => {
                                    const handleResize = (colKey: string, delta: number) => {
                                        setColumnWidths(prev => ({
                                            ...prev,
                                            [colKey]: Math.max(30, prev[colKey] + delta)
                                        }));
                                    };

                                    const Resizer = ({ colKey }: { colKey: string }) => (
                                        <div 
                                            className="resizer" 
                                            onMouseDown={(e) => {
                                                e.preventDefault();
                                                const startX = e.clientX;
                                                const startWidth = columnWidths[colKey];
                                                
                                                const onMouseMove = (moveEvent: MouseEvent) => {
                                                    const delta = moveEvent.clientX - startX;
                                                    setColumnWidths(prev => ({ ...prev, [colKey]: Math.max(30, startWidth + delta) }));
                                                };
                                                
                                                const onMouseUp = () => {
                                                    document.removeEventListener('mousemove', onMouseMove);
                                                    document.removeEventListener('mouseup', onMouseUp);
                                                };
                                                
                                                document.addEventListener('mousemove', onMouseMove);
                                                document.addEventListener('mouseup', onMouseUp);
                                            }}
                                            onClick={(e) => e.stopPropagation()}
                                        />
                                    );

                                    return (
                                        <tr style={{ background: 'var(--bg-dark)' }}>
                                            {showLineNumbers && (
                                                <th style={{ width: compactMode ? columnWidths.lineCompact : columnWidths.line, position: 'relative' }} className={compactMode ? "col-line-expanded" : "col-line"}>
                                                    Line <Resizer colKey={compactMode ? "lineCompact" : "line"} />
                                                </th>
                                            )}
                                            {showDate && (
                                                <th style={{ width: compactMode ? columnWidths.dateCompact : columnWidths.date, position: 'relative' }} className={compactMode ? "col-date-expanded" : "col-date"}>
                                                    Date <Resizer colKey={compactMode ? "dateCompact" : "date"} />
                                                </th>
                                            )}
                                            <th style={{ width: compactMode ? columnWidths.timeCompact : columnWidths.time, position: 'relative' }} className={compactMode ? "col-time-expanded" : "col-time"}>
                                                Time <Resizer colKey={compactMode ? "timeCompact" : "time"} />
                                            </th>
                                            {showLevel && (
                                                <th style={{ width: columnWidths.level, position: 'relative' }} className="col-level">
                                                    Level <Resizer colKey="level" />
                                                </th>
                                            )}
                                            <th style={{ width: columnWidths.tag, position: 'relative' }} className="col-tag">
                                                Tag <Resizer colKey="tag" />
                                            </th>
                                            <th className="col-msg">Message</th>
                                            {compactMode && (
                                                <th style={{ width: columnWidths.count, position: 'relative' }} className="col-count">
                                                    Count <Resizer colKey="count" />
                                                </th>
                                            )}
                                        </tr>
                                    );
                                }}
                                itemContent={(index, log: any) => {
                                    const formatTime = (time: string, ms: string, tz: string) => {
                                        let res = time || '';
                                        if (showMilliseconds && ms) res += `.${ms}`;
                                        if (showTimezone && tz) res += tz;
                                        return res;
                                    };
                                    
                                    const startDate = log.startDate || log.date;
                                    const endDate = log.endDate || log.date;
                                    const startStr = formatTime(log.startTime || log.time, log.startMs || log.ms, log.startTz || log.tz);
                                    const endStr = formatTime(log.endTime || log.time, log.endMs || log.ms, log.endTz || log.tz);

                                    return (
                                        <>
                                            {showLineNumbers && (
                                                <td style={{ width: compactMode ? columnWidths.lineCompact : columnWidths.line }} className={compactMode ? "col-line-expanded" : "col-line"}>
                                                    {compactMode && log.count > 1 ? `${log.startLine} - ${log.endLine}` : log.lineNum}
                                                </td>
                                            )}
                                            {showDate && (
                                                <td style={{ width: compactMode ? columnWidths.dateCompact : columnWidths.date }} className={compactMode ? "col-date-expanded" : "col-date"}>
                                                    {compactMode && log.count > 1 ? (startDate === endDate ? startDate : `${startDate} - ${endDate}`) : startDate}
                                                </td>
                                            )}
                                            <td style={{ width: compactMode ? columnWidths.timeCompact : columnWidths.time }} className={compactMode ? "col-time-expanded" : "col-time"}>
                                                {compactMode && log.count > 1 ? `${startStr} - ${endStr}` : startStr}
                                            </td>
                                            {showLevel && (
                                                <td style={{ width: columnWidths.level }} className="col-level">
                                                    {log.level ? (
                                                        <span className={`level-badge level-${log.level.toLowerCase()}`}>
                                                            {log.level}
                                                        </span>
                                                    ) : ''}
                                                </td>
                                            )}
                                            <td style={{ width: columnWidths.tag }} className="col-tag">{log.tag}</td>
                                            <td className="col-msg">
                                                <MessageCell text={log.message || log.raw} />
                                            </td>
                                            {compactMode && <td style={{ width: columnWidths.count }} className="col-count">{log.count > 1 ? log.count : ''}</td>}
                                        </>
                                    );
                                }}
                            />
                        </div>
                    </>
                ) : (
                    <div className="empty-state">
                        Select a file to view logs
                    </div>
                )}
            </div>
        </div>
        
            {/* Update Modal */}
            {showUpdateModal && updateInfo && (
                <div className="modal-overlay" onClick={() => !isUpdating && setShowUpdateModal(false)}>
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{ width: '500px' }}>
                        <h2>Update Available: {updateInfo.version}</h2>
                        <div style={{ marginTop: '15px', marginBottom: '20px', maxHeight: '300px', overflowY: 'auto', background: 'var(--bg-dark)', padding: '10px', borderRadius: '4px', whiteSpace: 'pre-wrap', fontSize: '13px' }}>
                            {updateInfo.releaseNotes || "No release notes provided."}
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                            <button 
                                onClick={() => setShowUpdateModal(false)} 
                                disabled={isUpdating}
                                style={{ padding: '8px 16px', background: 'var(--bg-lighter)', color: 'var(--text-color)', border: 'none', borderRadius: '4px', cursor: 'pointer' }}
                            >
                                Cancel
                            </button>
                            <button 
                                onClick={handleApplyUpdate} 
                                disabled={isUpdating}
                                style={{ padding: '8px 16px', background: 'var(--accent)', color: '#fff', border: 'none', borderRadius: '4px', cursor: isUpdating ? 'wait' : 'pointer' }}
                            >
                                {isUpdating ? 'Downloading & Updating...' : 'Update Now'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
}

export default App;
