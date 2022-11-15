import React, { useState } from "react";
import { Modal, SceneSelect } from "src/components/Shared";
import { useToast } from "src/hooks";
import { useIntl } from "react-intl";
import { faSignOutAlt } from "@fortawesome/free-solid-svg-icons";
import { Col, Form, Row } from "react-bootstrap";
import { FormUtils } from "src/utils";
import { mutateSceneAssignFile } from "src/core/StashService";

interface IFile {
  id: string;
  path: string;
}

interface IReassignFilesDialogProps {
  selected: IFile;
  onClose: () => void;
}

export const ReassignFilesDialog: React.FC<IReassignFilesDialogProps> = (
  props: IReassignFilesDialogProps
) => {
  const [scenes, setScenes] = useState<{ id: string; title: string }[]>([]);

  const intl = useIntl();
  const singularEntity = intl.formatMessage({ id: "file" });
  const pluralEntity = intl.formatMessage({ id: "files" });

  const header = intl.formatMessage(
    { id: "dialogs.reassign_entity_title" },
    { count: 1, singularEntity, pluralEntity }
  );

  const toastMessage = intl.formatMessage(
    { id: "toast.reassign_past_tense" },
    { count: 1, singularEntity, pluralEntity }
  );

  const Toast = useToast();

  // Network state
  const [reassigning, setReassigning] = useState(false);

  async function onAccept() {
    if (!scenes.length) {
      return;
    }

    setReassigning(true);
    try {
      await mutateSceneAssignFile(scenes[0].id, props.selected.id);
      Toast.success({ content: toastMessage });
      props.onClose();
    } catch (e) {
      Toast.error(e);
      props.onClose();
    }
    setReassigning(false);
  }

  return (
    <Modal
      show
      icon={faSignOutAlt}
      header={header}
      accept={{
        onClick: onAccept,
        text: intl.formatMessage({ id: "actions.reassign" }),
      }}
      cancel={{
        onClick: () => props.onClose(),
        text: intl.formatMessage({ id: "actions.cancel" }),
        variant: "secondary",
      }}
      isRunning={reassigning}
    >
      <Form>
        <Form.Group controlId="dest" as={Row}>
          {FormUtils.renderLabel({
            title: intl.formatMessage({
              id: "dialogs.reassign_files.destination",
            }),
            labelProps: {
              column: true,
              sm: 3,
              xl: 12,
            },
          })}
          <Col sm={9} xl={12}>
            <SceneSelect
              selected={scenes}
              onSelect={(items) => setScenes(items)}
            />
          </Col>
        </Form.Group>
      </Form>
    </Modal>
  );
};

export default ReassignFilesDialog;
