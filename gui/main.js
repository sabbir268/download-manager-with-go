const { app, BrowserWindow, ipcMain } = require('electron')
const { exec } = require('child_process')
const contextMenu = require('electron-context-menu');

contextMenu({
    showSaveImageAs: true,
    showInspectElement: true,
});

let mainWindow

function createWindow() {
    mainWindow = new BrowserWindow({
        width: 800,
        height: 600,
        webPreferences: {
            nodeIntegration: true,
            contextIsolation: false
        }
    })

    mainWindow.loadFile('./index.html')

    mainWindow.on('closed', function() {
        mainWindow = null
    })
}

app.whenReady().then(createWindow)

app.on('window-all-closed', function() {
    if (process.platform !== 'darwin') app.quit()
})

app.on('activate', function() {
    if (mainWindow === null) createWindow()
})



ipcMain.on('download', (event, url) => {
    const command = `./downloader -url ${url}`
    const child = exec(command)

    child.stdout.on('data', (data) => {
        const progress = parseFloat(data)
        event.sender.send('progress', progress)
    })

    child.stderr.on('data', (data) => {
        console.error(data)
    })

    child.on('close', (code) => {
        console.log(`child process exited with code ${code}`)
    })
})