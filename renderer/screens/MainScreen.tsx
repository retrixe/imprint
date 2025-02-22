import {
  Button,
  DialogContent,
  DialogTitle,
  Modal,
  ModalClose,
  ModalDialog,
  Option,
  Select,
  Textarea,
  Typography,
} from '@mui/joy'
import styles from './MainScreen.module.scss'
import { useState } from 'react'

const MainScreen = ({
  file,
  setFile,
  device,
  setDevice,
  devices,
  setDialog,
}: {
  file: string
  setFile: React.Dispatch<React.SetStateAction<string>>
  device: string | null
  setDevice: React.Dispatch<React.SetStateAction<string | null>>
  devices: string[]
  setDialog: React.Dispatch<React.SetStateAction<string>>
}): React.JSX.Element => {
  const [confirm, setConfirm] = useState(false)
  const onFileInputChange: React.ChangeEventHandler<HTMLTextAreaElement> = event =>
    setFile(event.target.value.replace(/\n/g, ''))
  const onFlashClick = (): void => {
    if (device === null) return setDialog('Error: Select a device to flash the image to!')
    if (file === '') return setDialog('Error: Select a disk image to flash to device!')
    setConfirm(true)
  }
  const onFlashConfirm = (): void => {
    if (device === null || file === '') return
    setConfirm(false)
    globalThis.flash(file, device.split(' ')[1], +device.split(' ')[0])
  }

  return (
    <>
      <Modal open={confirm} onClose={() => setConfirm(false)}>
        <ModalDialog>
          <ModalClose variant='soft' />
          <DialogTitle>Do you want to continue?</DialogTitle>
          <DialogContent>
            This operation will WIPE ALL DATA from: {device?.substring(device.indexOf(' ') + 1)}.
          </DialogContent>
          <Button color='danger' onClick={onFlashConfirm}>
            Proceed
          </Button>
        </ModalDialog>
      </Modal>
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
          value={device}
          required
          onChange={(_, value) => setDevice(value)}
        >
          {devices.map(device => (
            <Option key={device} value={device}>
              {device.substring(device.indexOf(' ') + 1)}
            </Option>
          ))}
        </Select>
        <Button onClick={() => globalThis.refreshDevices()} variant='soft'>
          Refresh
        </Button>
      </div>

      <br />
      <div className={styles['flash-progress-container']}>
        {/* FIXME: Add Settings dialog to disable validation and toggle dark mode. */}
        <div className={styles['full-width']} />
        <Button onClick={onFlashClick}>Flash</Button>
      </div>
    </>
  )
}

export default MainScreen
