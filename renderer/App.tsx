import { useEffect, useState } from 'react'
import JSBI from 'jsbi'
import Dialog from './Dialog'
import * as styles from './App.module.scss'
import { Button, Option, Select, Textarea, Typography, useColorScheme } from '@mui/joy'

// TODO: Experiment with lg size for the UI.
const App = (): JSX.Element => {
  const [file, setFile] = useState('')
  const [speed, setSpeed] = useState('')
  const [dialog, setDialog] = useState('')
  const [confirm, setConfirm] = useState(false)
  const [devices, setDevices] = useState<string[]>([])
  const [fileSize, setFileSize] = useState(0)
  const [progress, setProgress] = useState<number | string | null>(null)
  const [selectedDevice, setSelectedDevice] = useState<string | null>(null)
  globalThis.setFileReact = setFile
  globalThis.setSpeedReact = setSpeed
  globalThis.setDialogReact = setDialog
  globalThis.setDevicesReact = setDevices
  globalThis.setProgressReact = setProgress
  globalThis.setFileSizeReact = setFileSize
  useColorScheme().setMode('light')
  useEffect(() => {
    globalThis.refreshDevices()
  }, [])
  useEffect(() => setSelectedDevice(null), [devices])

  const inProgress = typeof progress === 'number'
  useEffect(() => setConfirm(false), [inProgress])
  const onFlashButtonClick = (): void => {
    if (inProgress) {
      // FIXME: A dialog would be better.
      if (confirm) {
        setConfirm(false)
        globalThis.cancelFlash()
      } else setConfirm(true)
      return
    }
    setProgress(null)
    if (selectedDevice === null) return setDialog('Error: Select a device to flash the ISO to!')
    if (file === '') return setDialog('Error: Select an ISO to flash to a device!')
    if (JSBI.greaterThan(JSBI.BigInt(fileSize), JSBI.BigInt(selectedDevice.split(' ')[0]))) {
      return setDialog('Error: The ISO file is too big to fit on the selected drive!')
    }
    if (!confirm) return setConfirm(true)
    setConfirm(false)
    globalThis.flash(file, selectedDevice.split(' ')[1])
  }
  const onFileInputChange: React.ChangeEventHandler<HTMLTextAreaElement> = event =>
    setFile(event.target.value.replace(/\n/g, ''))

  const progressPercent = inProgress
    ? JSBI.divide(JSBI.multiply(JSBI.BigInt(progress), JSBI.BigInt(100)), JSBI.BigInt(fileSize))
    : JSBI.BigInt(0)
  return (
    <div className={styles.root}>
      <Dialog
        open={dialog !== ''}
        onClose={() => setDialog('')}
        message={dialog.startsWith('Error: ') ? dialog.substring(7) : dialog}
        error={dialog.startsWith('Error: ')}
      />
      <div className={styles.container}>
        <Typography>Step 1: Select the disk image (.iso, .img, etc) to flash.</Typography>
        <div className={styles['select-container']}>
          <Button variant='soft' onClick={() => globalThis.promptForFile()}>
            Select File
          </Button>
          <Textarea
            minRows={2}
            maxRows={2}
            required
            placeholder='Path to disk image'
            className={styles['full-width']}
            value={file}
            onChange={onFileInputChange}
          />
        </div>
        <br />
        <Typography>Step 2: Select the device to flash to.</Typography>
        <div className={styles['select-container']}>
          <Select
            className={styles['full-width']}
            placeholder='Select a device'
            value={selectedDevice}
            required
            onChange={(_, value) => setSelectedDevice(value)}
          >
            {devices.map(device => (
              <Option key={device} value={device}>
                {device.substr(device.indexOf(' ') + 1)}
              </Option>
            ))}
          </Select>
          <Button onClick={() => globalThis.refreshDevices()} variant='soft'>
            Refresh
          </Button>
        </div>

        <br />
        <div className={styles['flash-progress-container']}>
          {/* FIXME: Add Settings dialog to disable validation. */}
          {inProgress && (
            // FIXME: Move this to a dedicated screen.
            <Typography className={styles['full-width']}>
              Progress: {progressPercent.toString()}% | Speed: {speed}
            </Typography>
          )}
          <div className={styles['full-width']} />
          <Button onClick={onFlashButtonClick}>
            {confirm ? 'Confirm?' : inProgress ? 'Cancel' : 'Flash'}
          </Button>
          {typeof progress === 'string' && <span>{progress}</span>}
        </div>
      </div>
    </div>
  )
}

export default App
