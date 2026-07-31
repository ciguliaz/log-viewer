import { useState, useEffect, useRef } from 'react';
import { main } from '../../wailsjs/go/models';
import { ProcessDrop, SelectFolder, ListFiles } from '../../wailsjs/go/main/App';
import { EventsOn } from '../../wailsjs/runtime/runtime';

export type WorkspaceFolder = {
    path: string;
    name: string;
    files: main.FileInfo[];
    error?: string;
};

export function useWorkspaces() {
    const [workspaces, setWorkspaces] = useState<WorkspaceFolder[]>([]);
    const [expandedFolders, setExpandedFolders] = useState<Set<string>>(new Set());
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

    // Periodically poll workspaces
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
                    const prevNames = prev[i].files.map(f => f.name).join('|');
                    const updatedNames = updatedWorkspaces[i].files.map(f => f.name).join('|');
                    if (prevNames !== updatedNames) {
                        changed = true;
                        break;
                    }
                }
                return changed ? updatedWorkspaces : prev;
            });
        }, 3000);
        
        return () => clearInterval(interval);
    }, [workspaces]);

    // Wails drag and drop
    useEffect(() => {
        const preventDefault = (e: Event) => {
            e.preventDefault();
            e.stopPropagation();
        };
        window.addEventListener('dragover', preventDefault);
        window.addEventListener('drop', preventDefault);

        EventsOn('wails:file-drop', async (...args: any[]) => {
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
                            files: dropResult.files || [],
                            error: dropResult.error || undefined
                        }];
                    });
                    setExpandedFolders(prev => new Set(prev).add(dropResult.path));
                }
            }
        });

        return () => {
            window.removeEventListener('dragover', preventDefault);
            window.removeEventListener('drop', preventDefault);
        };
    }, []);

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

    const handleRemoveFolder = (e: React.MouseEvent, path: string) => {
        e.stopPropagation();
        setWorkspaces(prev => prev.filter(w => w.path !== path));
    };

    return {
        workspaces,
        expandedFolders,
        handleSelectFolder,
        handleAddFolder,
        toggleFolder,
        handleRemoveFolder
    };
}
