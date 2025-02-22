import { DialogContent, DialogTitle, Modal, ModalClose, ModalDialog } from '@mui/joy'
import { useEffect, useState } from 'react'
import styles from './App.module.scss'
import MainScreen from './screens/MainScreen'
import ProgressScreen from './screens/ProgressScreen'

// TODO: Experiment with lg size for the UI.
const App = (): React.JSX.Element => {
  // useColorScheme().setMode('dark')
  const [file, setFile] = useState('')
  const [device, setDevice] = useState<string | null>(null)
  const [devices, setDevices] = useState<string[]>([])
  globalThis.setFileReact = setFile
  globalThis.setDevicesReact = setDevices
  const [dialog, setDialog] = useState('')
  globalThis.setDialogReact = setDialog
  const [progress, setProgress] = useState<Progress | string | null>(null)
  globalThis.setProgressReact = setProgress
  useEffect(() => {
    globalThis.refreshDevices()
  }, [])
  useEffect(() => setDevice(null), [devices])

  return (
    <div className={styles.root}>
      <Modal open={dialog !== ''} onClose={() => setDialog('')}>
        <ModalDialog color={dialog.startsWith('Error: ') ? 'danger' : undefined}>
          <ModalClose variant='soft' />
          <DialogTitle>{dialog.startsWith('Error: ') ? 'Error' : 'Info'}</DialogTitle>
          <DialogContent>
            {dialog.startsWith('Error: ') ? dialog.substring(7) : dialog}
          </DialogContent>
        </ModalDialog>
      </Modal>
      <div className={styles.container}>
        {progress === null && (
          <MainScreen
            file={file}
            setFile={setFile}
            device={device}
            setDevice={setDevice}
            devices={devices}
            setDialog={setDialog}
          />
        )}
        {progress !== null && (
          <ProgressScreen
            device={device ?? ''}
            file={file}
            progress={progress}
            onExit={() => {
              setFile('')
              setDevice(null)
              setProgress(null)
              globalThis.refreshDevices()
            }}
          />
        )}
      </div>
    </div>
  )
}

export default App
