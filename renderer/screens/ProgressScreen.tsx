import {
  Button,
  DialogContent,
  DialogTitle,
  LinearProgress,
  Modal,
  ModalClose,
  ModalDialog,
  Typography,
} from '@mui/joy'
import JSBI from 'jsbi'
import * as styles from './ProgressScreen.module.scss'
import { useEffect, useState } from 'react'

function bytesToString(bytes: number, binaryPowers = false): string {
  const divisor = binaryPowers ? 1024 : 1000
  const suffix = binaryPowers ? 'i' : ''

  const kb = bytes / divisor
  const mb = kb / divisor
  const gb = mb / divisor
  const tb = gb / divisor

  if (tb >= 1) {
    return `${tb.toFixed(1)} T${suffix}B`
  } else if (gb >= 1) {
    return `${gb.toFixed(1)} G${suffix}B`
  } else if (mb >= 1) {
    return `${mb.toFixed(1)} M${suffix}B`
  } else if (kb >= 1) {
    return `${kb.toFixed(1)} K${suffix}B`
  } else {
    return `${bytes}B`
  }
}

const ProgressScreen = ({
  progress,
  device,
  file,
  onExit,
}: {
  progress: Progress | string
  device: string
  file: string
  onExit: () => void
}): JSX.Element => {
  const [confirm, setConfirm] = useState(false)

  const isDone = progress === 'Done!'
  const isError = typeof progress === 'string' && !isDone
  const progressPercent =
    !isError && !isDone
      ? JSBI.divide(
          JSBI.multiply(JSBI.BigInt(progress.bytes), JSBI.BigInt(100)),
          JSBI.BigInt(progress.total),
        )
      : JSBI.BigInt(0)

  const sourceImage = file.replace('\\', '/').split('/').pop()
  const targetDisk = device.substr(device.indexOf(' ') + 1)
  const onDismiss = (): void => {
    if (isError || isDone) onExit()
    else setConfirm(true)
  }
  useEffect(() => {
    if (isDone || isError) setConfirm(false)
  }, [isDone, isError])

  return (
    <>
      <Modal open={confirm} onClose={() => setConfirm(false)}>
        <ModalDialog color='danger'>
          <ModalClose variant='soft' />
          <DialogTitle>Do you want to cancel flashing?</DialogTitle>
          <DialogContent>
            This will render the device {device?.substr(device.indexOf(' ') + 1)} unusable.
            <br />
            You must reformat the device to use it again.
          </DialogContent>
          <Button color='danger' onClick={() => globalThis.cancelFlash()}>
            Yes, I want to cancel
          </Button>
        </ModalDialog>
      </Modal>
      <Typography gutterBottom level='h3' color={isError ? 'danger' : undefined}>
        {isDone && 'Completed flashing ISO to disk!'}
        {isError && 'Error during '}
        {!isDone && 'Phase 1/1: Writing ISO to disk...'}
      </Typography>
      <LinearProgress
        sx={{ mb: '0.8em' }}
        color={isError ? 'danger' : undefined}
        determinate={!isError}
        value={isError ? undefined : isDone ? 100 : JSBI.toNumber(progressPercent)}
      />
      {!isDone && (
        <Typography level='title-lg' gutterBottom color={isError ? 'danger' : undefined}>
          {isError || isDone
            ? progress
            : `${progressPercent.toString()}% \
(${bytesToString(progress.bytes)} / ${bytesToString(progress.total)}) — ${progress.speed}`}
        </Typography>
      )}
      <Typography gutterBottom>
        <strong>Source Image:</strong> {sourceImage}
        <br />
        <strong>Target Disk:</strong> {targetDisk}
      </Typography>
      {!isError && !isDone && (
        <Typography gutterBottom color='warning'>
          <strong>Note:</strong> Do not remove the external drive or shut down the computer during
          the write process.
        </Typography>
      )}
      {isDone && (
        <Typography gutterBottom>
          <strong>Note:</strong> You may now remove the external drive safely.
        </Typography>
      )}
      <div className={styles['action-container']}>
        <div className={styles['full-width']} />
        <Button onClick={onDismiss} color={isDone ? 'primary' : 'danger'} variant='soft'>
          {isError || isDone ? 'Dismiss' : 'Cancel Flash'}
        </Button>
      </div>
    </>
  )
}

export default ProgressScreen
