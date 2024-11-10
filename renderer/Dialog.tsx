import { DialogContent, DialogTitle, Modal, ModalClose, ModalDialog } from '@mui/joy'
import * as styles from './Dialog.module.scss'

const Dialog = (props: {
  open: boolean
  onClose: () => void
  message: string
  error: boolean
}): JSX.Element => {
  return (
    <Modal open={props.open} onClose={props.onClose}>
      <ModalDialog className={styles['dialog-sheet']} color={props.error ? 'danger' : undefined}>
        <ModalClose variant='soft' />
        <DialogTitle>{props.error ? 'Error' : 'Info'}</DialogTitle>
        <DialogContent>{props.message}</DialogContent>
      </ModalDialog>
    </Modal>
  )
}

export default Dialog
