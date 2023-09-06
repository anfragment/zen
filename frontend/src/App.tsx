import {useState} from 'react';
import './App.css';
import { StartProxy, StopProxy } from '../wailsjs/go/main/App';

function App() {
    return (
        <div id="App">
            <button onClick={StartProxy}>Start</button>
            <button onClick={StopProxy}>Stop</button>
        </div>
    )
}

export default App
