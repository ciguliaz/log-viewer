import { useState, useEffect } from 'react';
import './App.css';
import { EventsOn } from '../wailsjs/runtime/runtime';
import { GetInitialConnections } from '../wailsjs/go/main/App';

interface Connection {
    app: string;
    hash: string;
    dest: string;
    packets: number;
    route: string;
    last_seen: string;
}

function App() {
    const [connections, setConnections] = useState<Connection[]>([]);

    useEffect(() => {
        // Fetch initial data
        GetInitialConnections().then(setConnections);

        // Listen for live updates
        EventsOn('connections_update', (data: Connection[]) => {
            // Sort by packets descending
            const sorted = [...data].sort((a, b) => b.packets - a.packets);
            setConnections(sorted);
        });
    }, []);

    return (
        <div className="dashboard">
            <div className="header">
                <h1>
                    🚀 Hatacone Shadow Live 
                    <span className="badge">{connections.length} Active</span>
                </h1>
            </div>
            
            <div className="table-container">
                <table>
                    <thead>
                        <tr>
                            <th>App</th>
                            <th>Hash</th>
                            <th>Destination</th>
                            <th>Route</th>
                            <th>Packets</th>
                            <th>Last Seen</th>
                        </tr>
                    </thead>
                    <tbody>
                        {connections.map((conn) => (
                            <tr key={conn.hash}>
                                <td>{conn.app}</td>
                                <td className="hash-cell">{conn.hash}</td>
                                <td>{conn.dest}</td>
                                <td>
                                    <span className={`route-badge route-${conn.route.toLowerCase()}`}>
                                        {conn.route.toUpperCase()}
                                    </span>
                                </td>
                                <td>
                                    <span className="packet-count" key={`${conn.hash}-${conn.packets}`}>
                                        {conn.packets.toLocaleString()}
                                    </span>
                                </td>
                                <td>{conn.last_seen}</td>
                            </tr>
                        ))}
                        {connections.length === 0 && (
                            <tr>
                                <td colSpan={6} style={{ textAlign: 'center', padding: '3rem', color: 'var(--text-muted)' }}>
                                    Waiting for shadow.log data...
                                </td>
                            </tr>
                        )}
                    </tbody>
                </table>
            </div>
        </div>
    );
}

export default App;
