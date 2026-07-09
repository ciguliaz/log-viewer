import { useState, useEffect, useMemo, useRef } from 'react';
import './App.css';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { SelectFolder, ListFiles, StartTailing, LoadPreviousChunk } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { TableVirtuoso, TableVirtuosoHandle } from 'react-virtuoso';

function App() {
    type WorkspaceFolder = {
        path: string;
        name: string;
        files: main.FileInfo[];
    };
    const [workspaces, setWorkspaces] = useState<WorkspaceFolder[]>([]);
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
    
    const [activeFile, setActiveFile] = useState<main.FileInfo | null>(null);
    const [logs, setLogs] = useState<main.LogEntry[]>([]);
    const [compactMode, setCompactMode] = useState<boolean>(false);
    const [showLineNumbers, setShowLineNumbers] = useState<boolean>(true);
    const [searchTerm, setSearchTerm] = useState<string>('');
    const [autoScroll, setAutoScroll] = useState<boolean>(true);
    const [firstItemIndex, setFirstItemIndex] = useState<number>(1000000);
    const [isLoadingHistory, setIsLoadingHistory] = useState<boolean>(false);
    const isLoadingMore = useRef(false);
    const activeFilePathRef = useRef<string | null>(null);
    
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
                    startTime: log.time,
                    endTime: log.time 
                };
                compacted.push(current);
            } else {
                if (current.tag === log.tag && current.message === log.message) {
                    current.count++;
                    if (log.time) current.endTime = log.time;
                    current.endLine = log.lineNum;
                } else {
                    current = { 
                        ...log, 
                        count: 1, 
                        startLine: log.lineNum, 
                        endLine: log.lineNum,
                        startTime: log.time,
                        endTime: log.time 
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
                files: fileList || []
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
                files: fileList || []
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

    const handleSelectFile = async (file: main.FileInfo) => {
        activeFilePathRef.current = file.path; // Instantly abort any active loops
        
        setActiveFile(file);
        setLogs([]); // Clear while loading
        setIsLoadingHistory(false);
        setAutoScroll(true); // Re-enable auto-scroll on new file
        setFirstItemIndex(1000000); // Reset index for new file
        
        await StartTailing(file.path);
        
        const currentPath = file.path;
        const initial = await LoadPreviousChunk();
        
        if (activeFilePathRef.current !== currentPath) return; // User clicked another file while waiting
        
        setLogs(initial || []);
        setIsLoadingHistory(true); // Start auto-loading the rest of the file
    };

    // Auto-loader for background history
    useEffect(() => {
        if (!isLoadingHistory || !activeFile) return;
        
        let active = true;
        const currentPath = activeFile.path;
        
        const loadRemaining = async () => {
            while (active && activeFilePathRef.current === currentPath) {
                if (isLoadingMore.current) {
                    await new Promise(r => setTimeout(r, 50));
                    continue;
                }
                
                isLoadingMore.current = true;
                const chunk = await LoadPreviousChunk();
                isLoadingMore.current = false;
                
                if (!active || activeFilePathRef.current !== currentPath) break;
                
                if (!chunk || chunk.length === 0) {
                    setIsLoadingHistory(false);
                    break;
                }
                
                setLogs(prev => [...chunk, ...prev]);
                setFirstItemIndex(prev => prev - chunk.length);
                
                // Yield to browser to prevent UI freeze
                await new Promise(r => setTimeout(r, 30));
            }
        };
        
        loadRemaining();
        
        return () => { active = false; };
    }, [isLoadingHistory, activeFile]);

    const loadMoreHistory = async () => {
        // If the auto-loader is already running, let it do the work
        if (isLoadingHistory || isLoadingMore.current) return;
        
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
                let next = [...prev];
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
                                >
                                    <span style={{fontSize: '9px'}}>{expandedFolders.has(ws.path) ? '▼' : '▶'}</span> {ws.name.toUpperCase()}
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
                                fixedHeaderContent={() => (
                                    <tr style={{ background: 'var(--bg-dark)' }}>
                                        {showLineNumbers && <th className={compactMode ? "col-line-expanded" : "col-line"}>Line</th>}
                                        <th className={compactMode ? "col-time-expanded" : "col-time"}>Time</th>
                                        <th className="col-tag">Tag</th>
                                        <th className="col-msg">Message</th>
                                        {compactMode && <th className="col-count">Count</th>}
                                    </tr>
                                )}
                                itemContent={(index, log) => (
                                    <>
                                        {showLineNumbers && (
                                            <td className={compactMode ? "col-line-expanded" : "col-line"}>
                                                {compactMode && log.count > 1 ? `${log.startLine} - ${log.endLine}` : log.lineNum}
                                            </td>
                                        )}
                                        <td className={compactMode ? "col-time-expanded" : "col-time"}>
                                            {compactMode && log.count > 1 ? `${log.startTime} - ${log.endTime}` : log.time}
                                        </td>
                                        <td className="col-tag">{log.tag}</td>
                                        <td className="col-msg">{log.message || log.raw}</td>
                                        {compactMode && <td className="col-count">{log.count > 1 ? log.count : ''}</td>}
                                    </>
                                )}
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
        </div>
    );
}

export default App;
