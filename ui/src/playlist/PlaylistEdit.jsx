import {
  Edit,
  FormDataConsumer,
  SimpleForm,
  TextInput,
  TextField,
  BooleanInput,
  required,
  useTranslate,
  usePermissions,
  ReferenceInput,
  SelectInput,
  useRecordContext,
  Toolbar,
  SaveButton,
  DeleteButton,
} from 'react-admin'
import { isWritable, Title } from '../common'
import SmartPlaylistFields from './SmartPlaylistFields'
import { buildPlaylistPayload, parseCriteriaToForm } from './smartPlaylistUtils'

const PlaylistEditToolbar = (props) => (
  <Toolbar {...props}>
    <SaveButton transform={buildPlaylistPayload} />
    <DeleteButton />
  </Toolbar>
)

const SyncFragment = ({ formData, variant, ...rest }) => {
  return (
    <>
      {formData.path && <BooleanInput source="sync" {...rest} />}
      {formData.path && <TextField source="path" {...rest} />}
    </>
  )
}

const PlaylistTitle = ({ record }) => {
  const translate = useTranslate()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  return <Title subTitle={`${resourceName} "${record ? record.name : ''}"`} />
}

const PlaylistEditForm = (props) => {
  const record = useRecordContext()
  const { permissions } = usePermissions()
  const smartDefaults = parseCriteriaToForm(record?.rules)
  return (
    <SimpleForm
      redirect="list"
      variant={'outlined'}
      {...props}
      initialValues={{ smart: !!record?.rules, ...smartDefaults }}
      toolbar={<PlaylistEditToolbar />}
    >
      <TextInput source="name" validate={required()} />
      <TextInput multiline source="comment" />
      {permissions === 'admin' ? (
        <ReferenceInput
          source="ownerId"
          reference="user"
          perPage={0}
          sort={{ field: 'name', order: 'ASC' }}
        >
          <SelectInput
            label={'resources.playlist.fields.ownerName'}
            optionText="userName"
          />
        </ReferenceInput>
      ) : (
        <TextField source="ownerName" />
      )}
      <BooleanInput source="public" disabled={!isWritable(record?.ownerId)} />
      <SmartPlaylistFields />
      <FormDataConsumer>
        {(formDataProps) => <SyncFragment {...formDataProps} />}
      </FormDataConsumer>
    </SimpleForm>
  )
}

const PlaylistEdit = (props) => (
  <Edit title={<PlaylistTitle />} actions={false} {...props}>
    <PlaylistEditForm {...props} />
  </Edit>
)

export default PlaylistEdit
