import { useState, useEffect } from 'react';
import './App.css';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { SelectFolder, ListFiles, StartTailing, GetInitialLogs } from '../wailsjs/go/main/App';
import { main } from '../wailsjs/go/models';

function App() {
    const [folder, setFolder] = useState<string>('');
    const [files, setFiles] = useState<main.FileInfo[]>([]);
    const [activeFile, setActiveFile] = useState<main.FileInfo | null>(null);
    const [logs, setLogs] = useState<main.LogEntry[]>([]);

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
        // Listen for live updates
        EventsOn('log_update', (data: main.LogEntry[]) => {
            setLogs(data || []);
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
                        <div className="menu-dropdown-item" style={{ color: 'var(--text-dim)' }}>
                            (Coming soon)
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
                        </div>
                        <div className="log-container">
                            <table>
                                <thead>
                                    <tr>
                                        <th className="col-time">Time</th>
                                        <th className="col-tag">Tag</th>
                                        <th className="col-msg">Message</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    {logs.map((log) => (
                                        <tr key={log.id}>
                                            <td className="col-time">{log.time}</td>
                                            <td className="col-tag">{log.tag}</td>
                                            <td className="col-msg">{log.message || log.raw}</td>
                                        </tr>
                                    ))}
                                </tbody>
                            </table>
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
