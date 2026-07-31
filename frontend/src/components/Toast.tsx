import React from 'react';

export type ToastMessage = {
    message: string;
    type: 'success' | 'error';
};

interface ToastProps {
    toast: ToastMessage | null;
}

export const Toast: React.FC<ToastProps> = ({ toast }) => {
    if (!toast) return null;

    return (
        <div style={{
            position: 'fixed',
            bottom: '20px',
            right: '20px',
            background: toast.type === 'success' ? '#10b981' : '#ef4444',
            color: '#fff',
            padding: '12px 20px',
            borderRadius: '6px',
            boxShadow: '0 4px 12px rgba(0,0,0,0.3)',
            zIndex: 2000,
            fontSize: '13px',
            fontWeight: 'bold',
            display: 'flex',
            alignItems: 'center',
            animation: 'slideIn 0.2s ease-out'
        }}>
            {toast.message}
        </div>
    );
};
