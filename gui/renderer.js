const { ipcRenderer } = require('electron')

document.getElementById('download').addEventListener('click', () => {
    const url = document.getElementById('url').value
    ipcRenderer.send('download', url)
})

ipcRenderer.on('progress', (event, progress) => {
    document.getElementById('progress').value = progress
})