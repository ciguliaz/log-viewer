import React, { useRef, useEffect } from 'react';
import { TableVirtuoso, TableVirtuosoHandle } from 'react-virtuoso';
import { models } from '../../wailsjs/go/models';
import { MessageCell } from './MessageCell';

interface LogViewerProps {
    activeFile: models.FileInfo | null;
    displayLogs: any[];
    searchTerm: string;
    setSearchTerm: (term: string) => void;
    autoScroll: boolean;
    setAutoScroll: (val: boolean) => void;
    firstItemIndex: number;
    loadMoreHistory: () => void;
    
    compactMode: boolean;
    showLineNumbers: boolean;
    showDate: boolean;
    showLevel: boolean;
    showMilliseconds: boolean;
    showTimezone: boolean;
    columnWidths: Record<string, number>;
    setColumnWidths: React.Dispatch<React.SetStateAction<Record<string, number>>>;
}

export const LogViewer: React.FC<LogViewerProps> = ({
    activeFile,
    displayLogs,
    searchTerm,
    setSearchTerm,
    autoScroll,
    setAutoScroll,
    firstItemIndex,
    loadMoreHistory,
    compactMode,
    showLineNumbers,
    showDate,
    showLevel,
    showMilliseconds,
    showTimezone,
    columnWidths,
    setColumnWidths
}) => {
    const virtuosoRef = useRef<TableVirtuosoHandle>(null);

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

    if (!activeFile) {
        return (
            <div className="main-view">
                <div className="empty-state">
                    Select a file to view logs
                </div>
            </div>
        );
    }

    return (
        <div className="main-view">
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
                    atBottomStateChange={(atBottom) => setAutoScroll(atBottom)}
                    fixedHeaderContent={() => {
                        const Resizer = ({ colKey }: { colKey: string }) => (
                            <div 
                                className="resizer" 
                                onMouseDown={(e) => {
                                    e.preventDefault();
                                    const startX = e.clientX;
                                    const startWidth = columnWidths[colKey];
                                    
                                    const onMouseMove = (moveEvent: MouseEvent) => {
                                        const delta = moveEvent.clientX - startX;
                                        setColumnWidths(prev => ({ ...prev, [colKey]: Math.max(30, startWidth + delta) }));
                                    };
                                    
                                    const onMouseUp = () => {
                                        document.removeEventListener('mousemove', onMouseMove);
                                        document.removeEventListener('mouseup', onMouseUp);
                                    };
                                    
                                    document.addEventListener('mousemove', onMouseMove);
                                    document.addEventListener('mouseup', onMouseUp);
                                }}
                                onClick={(e) => e.stopPropagation()}
                            />
                        );

                        return (
                            <tr style={{ background: 'var(--bg-dark)' }}>
                                {showLineNumbers && (
                                    <th style={{ width: compactMode ? columnWidths.lineCompact : columnWidths.line, position: 'relative' }} className={compactMode ? "col-line-expanded" : "col-line"}>
                                        Line <Resizer colKey={compactMode ? "lineCompact" : "line"} />
                                    </th>
                                )}
                                {showDate && (
                                    <th style={{ width: compactMode ? columnWidths.dateCompact : columnWidths.date, position: 'relative' }} className={compactMode ? "col-date-expanded" : "col-date"}>
                                        Date <Resizer colKey={compactMode ? "dateCompact" : "date"} />
                                    </th>
                                )}
                                <th style={{ width: compactMode ? columnWidths.timeCompact : columnWidths.time, position: 'relative' }} className={compactMode ? "col-time-expanded" : "col-time"}>
                                    Time <Resizer colKey={compactMode ? "timeCompact" : "time"} />
                                </th>
                                {showLevel && (
                                    <th style={{ width: columnWidths.level, position: 'relative' }} className="col-level">
                                        Level <Resizer colKey="level" />
                                    </th>
                                )}
                                <th style={{ width: columnWidths.tag, position: 'relative' }} className="col-tag">
                                    Tag <Resizer colKey="tag" />
                                </th>
                                <th className="col-msg">Message</th>
                                {compactMode && (
                                    <th style={{ width: columnWidths.count, position: 'relative' }} className="col-count">
                                        Count <Resizer colKey="count" />
                                    </th>
                                )}
                            </tr>
                        );
                    }}
                    itemContent={(_, log: any) => {
                        const formatTime = (time: string, ms: string, tz: string) => {
                            let res = time || '';
                            if (showMilliseconds && ms) res += `.${ms}`;
                            if (showTimezone && tz) res += tz;
                            return res;
                        };
                        
                        const startDate = log.startDate || log.date;
                        const endDate = log.endDate || log.date;
                        const startStr = formatTime(log.startTime || log.time, log.startMs || log.ms, log.startTz || log.tz);
                        const endStr = formatTime(log.endTime || log.time, log.endMs || log.ms, log.endTz || log.tz);

                        return (
                            <>
                                {showLineNumbers && (
                                    <td style={{ width: compactMode ? columnWidths.lineCompact : columnWidths.line }} className={compactMode ? "col-line-expanded" : "col-line"}>
                                        {compactMode && log.count > 1 ? `${log.startLine} - ${log.endLine}` : log.lineNum}
                                    </td>
                                )}
                                {showDate && (
                                    <td style={{ width: compactMode ? columnWidths.dateCompact : columnWidths.date }} className={compactMode ? "col-date-expanded" : "col-date"}>
                                        {compactMode && log.count > 1 ? (startDate === endDate ? startDate : `${startDate} - ${endDate}`) : startDate}
                                    </td>
                                )}
                                <td style={{ width: compactMode ? columnWidths.timeCompact : columnWidths.time }} className={compactMode ? "col-time-expanded" : "col-time"}>
                                    {compactMode && log.count > 1 ? `${startStr} - ${endStr}` : startStr}
                                </td>
                                {showLevel && (
                                    <td style={{ width: columnWidths.level }} className="col-level">
                                        {log.level ? (
                                            <span className={`level-badge level-${log.level.toLowerCase()}`}>
                                                {log.level}
                                            </span>
                                        ) : ''}
                                    </td>
                                )}
                                <td style={{ width: columnWidths.tag }} className="col-tag">{log.tag}</td>
                                <td className="col-msg">
                                    <MessageCell text={log.message || log.raw} />
                                </td>
                                {compactMode && <td style={{ width: columnWidths.count }} className="col-count">{log.count > 1 ? log.count : ''}</td>}
                            </>
                        );
                    }}
                />
            </div>
        </div>
    );
};
