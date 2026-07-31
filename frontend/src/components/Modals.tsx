import React from 'react';

interface ModalsProps {
    appVersion: string;
    
    updateInfo: any;
    showUpdateModal: boolean;
    setShowUpdateModal: (val: boolean) => void;
    isUpdating: boolean;
    handleApplyUpdate: () => void;
    
    showFeedbackModal: boolean;
    setShowFeedbackModal: (val: boolean) => void;
    feedbackType: 'Bug' | 'Feature';
    setFeedbackType: (val: 'Bug' | 'Feature') => void;
    feedbackText: string;
    setFeedbackText: (val: string) => void;
    isSubmittingFeedback: boolean;
    handleSubmitFeedback: () => void;
}

export const Modals: React.FC<ModalsProps> = ({
    appVersion,
    updateInfo, showUpdateModal, setShowUpdateModal, isUpdating, handleApplyUpdate,
    showFeedbackModal, setShowFeedbackModal, feedbackType, setFeedbackType, feedbackText, setFeedbackText, isSubmittingFeedback, handleSubmitFeedback
}) => {
    return (
        <>
            {/* Update Modal */}
            {showUpdateModal && updateInfo && (
                <div className="modal-overlay" onClick={() => !isUpdating && setShowUpdateModal(false)}>
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{ width: '450px' }}>
                        <h2>Update to {updateInfo.version}</h2>
                        <div style={{ marginTop: '12px', marginBottom: '16px', fontSize: '12px', color: 'var(--text-dim)' }}>
                            Current version: {appVersion}
                        </div>
                        <div style={{ marginBottom: '20px', maxHeight: '200px', overflowY: 'auto', background: 'var(--bg-dark)', padding: '10px', borderRadius: '4px', whiteSpace: 'pre-wrap', fontSize: '12px', color: 'var(--text-main)' }}>
                            {(updateInfo.releaseNotes || 'No release notes provided.').replace(/\*\*/g, '')}
                        </div>
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button 
                                onClick={() => setShowUpdateModal(false)} 
                                disabled={isUpdating}
                                style={{ padding: '6px 14px', background: 'var(--bg-hover)', color: 'var(--text-main)', border: '1px solid var(--border-color)', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}
                            >
                                Later
                            </button>
                            <button 
                                onClick={handleApplyUpdate} 
                                disabled={isUpdating}
                                style={{ padding: '6px 14px', background: 'var(--accent)', color: '#fff', border: 'none', borderRadius: '4px', cursor: isUpdating ? 'wait' : 'pointer', fontSize: '12px' }}
                            >
                                {isUpdating ? 'Updating...' : 'Update Now'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
            
            {/* Feedback Modal */}
            {showFeedbackModal && (
                <div className="modal-overlay" onClick={() => !isSubmittingFeedback && setShowFeedbackModal(false)}>
                    <div className="modal-content" onClick={e => e.stopPropagation()} style={{ width: '450px' }}>
                        <h2>{feedbackType === 'Bug' ? 'Report a Bug' : 'Suggest a Feature'}</h2>
                        <div style={{ marginTop: '12px', marginBottom: '16px', fontSize: '12px', color: 'var(--text-dim)' }}>
                            Help us improve Log Viewer. Your feedback will be sent directly to the developer.
                        </div>
                        
                        <div style={{ marginBottom: '12px' }}>
                            <label style={{ display: 'block', fontSize: '12px', marginBottom: '6px', color: 'var(--text-main)' }}>Type</label>
                            <select 
                                value={feedbackType} 
                                onChange={(e) => setFeedbackType(e.target.value as 'Bug' | 'Feature')}
                                style={{ width: '100%', padding: '8px', background: 'var(--bg-dark)', border: '1px solid var(--border-color)', color: 'var(--text-main)', borderRadius: '4px' }}
                            >
                                <option value="Bug">Bug Report</option>
                                <option value="Feature">Feature Request</option>
                            </select>
                        </div>
                        
                        <div style={{ marginBottom: '20px' }}>
                            <label style={{ display: 'block', fontSize: '12px', marginBottom: '6px', color: 'var(--text-main)' }}>Description</label>
                            <textarea 
                                value={feedbackText}
                                onChange={(e) => setFeedbackText(e.target.value)}
                                placeholder={feedbackType === 'Bug' ? 'Describe the bug, how to reproduce it, and what you expected to happen...' : 'Describe the feature you would like to see and why it would be useful...'}
                                style={{ width: '100%', height: '120px', padding: '8px', background: 'var(--bg-dark)', border: '1px solid var(--border-color)', color: 'var(--text-main)', borderRadius: '4px', resize: 'none', fontFamily: 'inherit' }}
                            />
                        </div>
                        
                        <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                            <button 
                                onClick={() => setShowFeedbackModal(false)} 
                                disabled={isSubmittingFeedback}
                                style={{ padding: '6px 14px', background: 'var(--bg-hover)', color: 'var(--text-main)', border: '1px solid var(--border-color)', borderRadius: '4px', cursor: 'pointer', fontSize: '12px' }}
                            >
                                Cancel
                            </button>
                            <button 
                                onClick={handleSubmitFeedback} 
                                disabled={isSubmittingFeedback || !feedbackText.trim()}
                                style={{ padding: '6px 14px', background: 'var(--accent)', color: '#fff', border: 'none', borderRadius: '4px', cursor: isSubmittingFeedback || !feedbackText.trim() ? 'not-allowed' : 'pointer', fontSize: '12px', opacity: !feedbackText.trim() ? 0.6 : 1 }}
                            >
                                {isSubmittingFeedback ? 'Submitting...' : 'Submit Feedback'}
                            </button>
                        </div>
                    </div>
                </div>
            )}
        </>
    );
};
