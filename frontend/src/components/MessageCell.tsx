import { useState } from 'react';

export const MessageCell = ({ text }: { text: string }) => {
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
