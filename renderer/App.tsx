import { useEffect, useState } from 'react'
import JSBI from 'jsbi'
import Dialog from './Dialog'
import * as styles from './App.module.scss'

const App = (): JSX.Element => {
  const [file, setFile] = useState('')
  const [speed, setSpeed] = useState('')
  const [dialog, setDialog] = useState('')
  const [confirm, setConfirm] = useState(false)
  const [devices, setDevices] = useState(['N/A'])
  const [fileSize, setFileSize] = useState(0)
  const [progress, setProgress] = useState<number | string | null>(null)
  const [selectedDevice, setSelectedDevice] = useState('N/A')
  globalThis.setFileReact = setFile
  globalThis.setSpeedReact = setSpeed
  globalThis.setDialogReact = setDialog
  globalThis.setDevicesReact = setDevices
  globalThis.setProgressReact = setProgress
  globalThis.setFileSizeReact = setFileSize
  globalThis.setSelectedDeviceReact = setSelectedDevice
  useEffect(() => {
    globalThis.refreshDevices()
  }, [])

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
    if (selectedDevice === 'N/A') return setDialog('Error: Select a device to flash the ISO to!')
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
    <>
      {dialog !== '' && (
        <Dialog
          handleDismiss={() => setDialog('')}
          message={dialog.startsWith('Error: ') ? dialog.substring(7) : dialog}
          error={dialog.startsWith('Error: ')}
        />
      )}
      <div className={styles.container}>
        <span>Step 1: Enter the path to the file.</span>
        <div className={styles['select-image-container']}>
          <textarea className={styles['full-width']} value={file} onChange={onFileInputChange} />
          <button onClick={() => globalThis.promptForFile()}>Select ISO</button>
        </div>
        <span>Step 2: Select the device to flash the ISO to.</span>
        <div className={styles['select-device-container']}>
          <select
            className={styles['full-width']}
            value={selectedDevice}
            onChange={e => setSelectedDevice(e.target.value)}
          >
            {devices.map(device => (
              <option key={device} value={device}>
                {device.substr(device.indexOf(' ') + 1)}
              </option>
            ))}
          </select>
          <button onClick={() => globalThis.refreshDevices()} className={styles['refresh-button']}>
            Refresh
          </button>
        </div>
        <span>Step 3: Click the button below to begin flashing.</span>
        <div className={styles['flash-progress-container']}>
          <button onClick={onFlashButtonClick}>
            {confirm ? 'Confirm?' : inProgress ? 'Cancel' : 'Flash'}
          </button>
          <div className={styles.spacer} />
          {inProgress && (
            <span>
              Progress: {progressPercent.toString()}% | Speed: {speed}
            </span>
          )}
          {typeof progress === 'string' && <span>{progress}</span>}
        </div>
      </div>
    </>
  )
}

export default App
