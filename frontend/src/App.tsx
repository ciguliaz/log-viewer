import './App.css';
import { useViewSettings } from './hooks/useViewSettings';
import { useWorkspaces } from './hooks/useWorkspaces';
import { useLogs } from './hooks/useLogs';
import { useUI } from './hooks/useUI';

import { MenuBar } from './components/MenuBar';
import { Sidebar } from './components/Sidebar';
import { LogViewer } from './components/LogViewer';
import { Modals } from './components/Modals';
import { Toast } from './components/Toast';

function App() {
    const viewSettings = useViewSettings();
    const workspaces = useWorkspaces();
    const logs = useLogs(viewSettings.compactMode);
    const ui = useUI();

    return (
        <div className="app-container">
            <MenuBar 
                handleSelectFolder={workspaces.handleSelectFolder}
                handleAddFolder={workspaces.handleAddFolder}
                compactMode={viewSettings.compactMode}
                setCompactMode={viewSettings.setCompactMode}
                showLineNumbers={viewSettings.showLineNumbers}
                setShowLineNumbers={viewSettings.setShowLineNumbers}
                showLevel={viewSettings.showLevel}
                setShowLevel={viewSettings.setShowLevel}
                showDate={viewSettings.showDate}
                setShowDate={viewSettings.setShowDate}
                showMilliseconds={viewSettings.showMilliseconds}
                setShowMilliseconds={viewSettings.setShowMilliseconds}
                showTimezone={viewSettings.showTimezone}
                setShowTimezone={viewSettings.setShowTimezone}
                setFeedbackType={ui.setFeedbackType}
                setShowFeedbackModal={ui.setShowFeedbackModal}
                updateInfo={ui.updateInfo}
                setShowUpdateModal={ui.setShowUpdateModal}
            />
            
            <div className="layout">
                <Sidebar 
                    workspaces={workspaces.workspaces}
                    expandedFolders={workspaces.expandedFolders}
                    activeFile={logs.activeFile}
                    appVersion={ui.appVersion}
                    isCheckingUpdate={ui.isCheckingUpdate}
                    toggleFolder={workspaces.toggleFolder}
                    handleRemoveFolder={(e, path) => {
                        workspaces.handleRemoveFolder(e, path);
                        logs.stopTailingIfActive(path);
                    }}
                    handleSelectFile={logs.handleSelectFile}
                    checkUpdate={ui.checkUpdate}
                />
                
                <LogViewer 
                    activeFile={logs.activeFile}
                    displayLogs={logs.displayLogs}
                    searchTerm={logs.searchTerm}
                    setSearchTerm={logs.setSearchTerm}
                    autoScroll={logs.autoScroll}
                    setAutoScroll={logs.setAutoScroll}
                    firstItemIndex={logs.firstItemIndex}
                    loadMoreHistory={logs.loadMoreHistory}
                    
                    compactMode={viewSettings.compactMode}
                    showLineNumbers={viewSettings.showLineNumbers}
                    showDate={viewSettings.showDate}
                    showLevel={viewSettings.showLevel}
                    showMilliseconds={viewSettings.showMilliseconds}
                    showTimezone={viewSettings.showTimezone}
                    columnWidths={viewSettings.columnWidths}
                    setColumnWidths={viewSettings.setColumnWidths}
                />
            </div>

            <Modals 
                appVersion={ui.appVersion}
                updateInfo={ui.updateInfo}
                showUpdateModal={ui.showUpdateModal}
                setShowUpdateModal={ui.setShowUpdateModal}
                isUpdating={ui.isUpdating}
                handleApplyUpdate={ui.handleApplyUpdate}
                
                showFeedbackModal={ui.showFeedbackModal}
                setShowFeedbackModal={ui.setShowFeedbackModal}
                feedbackType={ui.feedbackType}
                setFeedbackType={ui.setFeedbackType}
                feedbackText={ui.feedbackText}
                setFeedbackText={ui.setFeedbackText}
                isSubmittingFeedback={ui.isSubmittingFeedback}
                handleSubmitFeedback={ui.handleSubmitFeedback}
            />
            
            <Toast toast={ui.toast} />
        </div>
    );
}

export default App;
