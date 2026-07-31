import React from 'react';
import { WorkspaceFolder } from '../hooks/useWorkspaces';
import { main } from '../../wailsjs/go/models';

interface SidebarProps {
    workspaces: WorkspaceFolder[];
    expandedFolders: Set<string>;
    activeFile: main.FileInfo | null;
    appVersion: string;
    isCheckingUpdate: boolean;
    
    toggleFolder: (path: string) => void;
    handleRemoveFolder: (e: React.MouseEvent, path: string) => void;
    handleSelectFile: (file: main.FileInfo) => void;
    checkUpdate: (manual: boolean) => void;
}

export const Sidebar: React.FC<SidebarProps> = ({
    workspaces,
    expandedFolders,
    activeFile,
    appVersion,
    isCheckingUpdate,
    toggleFolder,
    handleRemoveFolder,
    handleSelectFile,
    checkUpdate
}) => {
    return (
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
            <div className="version-text" style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
                <span>{appVersion || '...'}</span>
                <span 
                    onClick={() => checkUpdate(true)} 
                    style={{ cursor: isCheckingUpdate ? 'wait' : 'pointer', color: 'var(--accent)' }}
                >
                    {isCheckingUpdate ? 'Checking...' : 'Check for updates'}
                </span>
            </div>
        </div>
    );
};
