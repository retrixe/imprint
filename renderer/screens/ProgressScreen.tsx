import { Button, LinearProgress, Typography } from '@mui/joy'
import JSBI from 'jsbi'
import * as styles from './ProgressScreen.module.scss'

const ProgressScreen = ({
  progress,
  setProgress,
  device,
  file,
}: {
  progress: Progress | string
  setProgress: React.Dispatch<React.SetStateAction<Progress | string | null>>
  device: string
  file: string
}): JSX.Element => {
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
    if (isError || isDone) setProgress(null)
    // Urgent FIXME: Cancel flash
  }

  return (
    <>
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
            ? progress // Urgent FIXME byte conversion...
            : `${progressPercent.toString()}% (${progress.bytes} bytes / ${progress.total} bytes) — ${progress.speed}`}
        </Typography>
      )}
      <Typography gutterBottom>
        <strong>Source Image: </strong> {sourceImage}
        <br />
        <strong>Target Disk: </strong> {targetDisk}
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
