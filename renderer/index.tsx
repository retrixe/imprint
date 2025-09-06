import { CssBaseline, CssVarsProvider, extendTheme } from '@mui/joy'
import { createRoot } from 'react-dom/client'
import App from './App'

declare global {
  // Exports from Go app process.
  var flash: (filePath: string, devicePath: string, deviceSize: number) => void
  var cancelFlash: () => void
  var promptForFile: () => void
  var refreshDevices: () => void
  // Export React state to the global scope.
  var setFileReact: (file: string) => void
  var setDevicesReact: (devices: string[]) => void
  var setDialogReact: (dialog: string) => void
  var setProgressReact: (progress: Progress | string | null) => void
  interface Progress {
    bytes: number
    total: number
    speed: string
    phase: string
  }
}

const theme = extendTheme({
  fontFamily: {
    body: 'system-ui, -apple-system, BlinkMacSystemFont, avenir next, avenir, segoe ui, helvetica neue, helvetica, Cantarell, Ubuntu, roboto, noto, arial, sans-serif',
    display:
      'system-ui, -apple-system, BlinkMacSystemFont, avenir next, avenir, segoe ui, helvetica neue, helvetica, Cantarell, Ubuntu, roboto, noto, arial, sans-serif',
  },
})

const el = document.getElementById('app')
if (el !== null) {
  createRoot(el).render(
    <CssVarsProvider theme={theme}>
      <CssBaseline />
      <App />
    </CssVarsProvider>,
  )
}
