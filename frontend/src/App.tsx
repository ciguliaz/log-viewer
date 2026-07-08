import { useState, useEffect, useMemo } from 'react';
import './App.css';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { SelectFolder, ListFiles, StartTailing, GetInitialLogs } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';
import { TableVirtuoso } from 'react-virtuoso';

function App() {
    const [folder, setFolder] = useState<string>('');
    const [files, setFiles] = useState<main.FileInfo[]>([]);
    const [activeFile, setActiveFile] = useState<main.FileInfo | null>(null);
    const [logs, setLogs] = useState<main.LogEntry[]>([]);
    const [compactMode, setCompactMode] = useState<boolean>(false);
    const [searchTerm, setSearchTerm] = useState<string>('');

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
        
        const mapped = result.map(l => ({
            ...l,
            startTime: l.time,
            endTime: l.time,
            count: 1
        }));

        if (!compactMode) {
            return mapped;
        }

        const compacted: any[] = [];
        let current: any = null;

        for (const log of mapped) {
            if (!current) {
                current = { ...log };
                compacted.push(current);
            } else {
                if (current.tag === log.tag && current.message === log.message) {
                    current.count++;
                    if (log.time) current.endTime = log.time;
                } else {
                    current = { ...log };
                    compacted.push(current);
                }
            }
        }
        return compacted;
    }, [logs, searchTerm, compactMode]);

    const handleSelectFolder = async () => {
        const selected = await SelectFolder();
        if (selected) {
            setFolder(selected);
            const fileList = await ListFiles(selected);
            setFiles(fileList || []);
        }
    };

    const handleSelectFile = async (file: main.FileInfo) => {
        setActiveFile(file);
        setLogs([]); // Clear while loading
        await StartTailing(file.path);
        const initial = await GetInitialLogs();
        setLogs(initial || []);
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

    return (
        <div className="app-container">
            <div className="menu-bar">
                <div className="menu-item">
                    File
                    <div className="menu-dropdown">
                        <div className="menu-dropdown-item" onClick={handleSelectFolder}>
                            Open Folder...
                        </div>
                    </div>
                </div>
                <div className="menu-item">
                    View
                    <div className="menu-dropdown">
                        <div className="menu-dropdown-item" onClick={() => setCompactMode(!compactMode)}>
                            <span style={{ width: '20px' }}>{compactMode ? '✓' : ''}</span> Compact Mode
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
                    {files.map(f => (
                        <div 
                            key={f.path} 
                            className={`file-item ${activeFile?.path === f.path ? 'active' : ''}`}
                            onClick={() => handleSelectFile(f)}
                        >
                            📄 {f.name}
                        </div>
                    ))}
                    {files.length === 0 && folder && (
                        <div style={{padding: '10px', color: 'var(--text-dim)'}}>No .log files found.</div>
                    )}
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
                                className="virtuoso-table"
                                data={displayLogs}
                                fixedHeaderContent={() => (
                                    <tr style={{ background: 'var(--bg-dark)' }}>
                                        <th className={compactMode ? "col-time-expanded" : "col-time"}>Time</th>
                                        <th className="col-tag">Tag</th>
                                        <th className="col-msg">Message</th>
                                        {compactMode && <th className="col-count">Count</th>}
                                    </tr>
                                )}
                                itemContent={(index, log) => (
                                    <>
                                        <td className={compactMode ? "col-time-expanded" : "col-time"}>
                                            {log.count > 1 ? `${log.startTime} - ${log.endTime}` : log.startTime}
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
