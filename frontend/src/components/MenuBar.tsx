import React, { useState, useEffect } from 'react';
import { WindowMinimise, WindowToggleMaximise, WindowIsMaximised, Quit, BrowserOpenURL } from '../../wailsjs/runtime/runtime';

interface MenuBarProps {
    handleSelectFolder: () => void;
    handleAddFolder: () => void;
    
    // View Settings
    compactMode: boolean; setCompactMode: (v: boolean) => void;
    showLineNumbers: boolean; setShowLineNumbers: (v: boolean) => void;
    showLevel: boolean; setShowLevel: (v: boolean) => void;
    showDate: boolean; setShowDate: (v: boolean) => void;
    showMilliseconds: boolean; setShowMilliseconds: (v: boolean) => void;
    showTimezone: boolean; setShowTimezone: (v: boolean) => void;
    
    // UI
    setFeedbackType: (type: 'Bug' | 'Feature') => void;
    setShowFeedbackModal: (show: boolean) => void;
    updateInfo: any;
    setShowUpdateModal: (show: boolean) => void;
}

export const MenuBar: React.FC<MenuBarProps> = ({
    handleSelectFolder, handleAddFolder,
    compactMode, setCompactMode,
    showLineNumbers, setShowLineNumbers,
    showLevel, setShowLevel,
    showDate, setShowDate,
    showMilliseconds, setShowMilliseconds,
    showTimezone, setShowTimezone,
    setFeedbackType, setShowFeedbackModal,
    updateInfo, setShowUpdateModal
}) => {
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

    return (
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
            <div className="menu-item">
                Help
                <div className="menu-dropdown">
                    <div className="menu-dropdown-item" onClick={() => { setFeedbackType('Bug'); setShowFeedbackModal(true); }}>
                        Report a Bug...
                    </div>
                    <div className="menu-dropdown-item" onClick={() => { setFeedbackType('Feature'); setShowFeedbackModal(true); }}>
                        Suggest a Feature...
                    </div>
                    <div style={{ height: '1px', background: 'var(--border-color)', margin: '4px 0' }}></div>
                    <div className="menu-dropdown-item" onClick={() => BrowserOpenURL("mailto:ckazzgd@gmail.com?subject=Log%20Viewer%20Feedback")}>
                        Email Support...
                    </div>
                </div>
            </div>
            
            <div className="window-controls">
                {updateInfo && (
                    <div 
                        onClick={() => setShowUpdateModal(true)}
                        style={{ cursor: 'pointer', color: 'var(--accent)', fontSize: '11px', marginRight: '12px', display: 'flex', alignItems: 'center', '--wails-draggable': 'no-drag' } as React.CSSProperties}
                    >
                        Update available
                    </div>
                )}
                <div className="window-control" onClick={() => WindowMinimise()}>
                    <svg width="12" height="12" viewBox="0 0 12 12"><rect fill="currentColor" width="10" height="1" x="1" y="6"></rect></svg>
                </div>
                <div className="window-control" onClick={() => WindowToggleMaximise()}>
                    {isMaximized ? (
                        <svg width="12" height="12" viewBox="0 0 12 12">
                            <path fill="currentColor" d="M3,3 H9 V9 H3 Z M4,4 V8 H8 V4 Z M2,2 H10 V10 H2 Z" fillRule="evenodd"></path>
                        </svg>
                    ) : (
                        <svg width="12" height="12" viewBox="0 0 12 12">
                            <rect width="9" height="9" x="1.5" y="1.5" fill="none" stroke="currentColor"></rect>
                        </svg>
                    )}
                </div>
                <div className="window-control close-control" onClick={() => Quit()}>
                    <svg width="12" height="12" viewBox="0 0 12 12">
                        <path fill="currentColor" d="M1.4,1.4 L10.6,10.6 M1.4,10.6 L10.6,1.4" stroke="currentColor" strokeWidth="1.2"></path>
                    </svg>
                </div>
            </div>
        </div>
    );
};
