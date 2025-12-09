import React from 'react'
import {
  Create,
  SimpleForm,
  TextInput,
  BooleanInput,
  required,
  useTranslate,
  useRefresh,
  useNotify,
  useRedirect,
  usePermissions,
  ReferenceInput,
  SelectInput,
  Toolbar,
  SaveButton,
} from 'react-admin'
import { Title } from '../common'
import SmartPlaylistFields from './SmartPlaylistFields'
import { buildPlaylistPayload } from './smartPlaylistUtils'

const PlaylistCreateToolbar = (props) => (
  <Toolbar {...props}>
    <SaveButton transform={buildPlaylistPayload} />
  </Toolbar>
)

const PlaylistCreate = (props) => {
  const { basePath } = props
  const refresh = useRefresh()
  const notify = useNotify()
  const redirect = useRedirect()
  const translate = useTranslate()
  const { permissions } = usePermissions()
  const resourceName = translate('resources.playlist.name', { smart_count: 1 })
  const title = translate('ra.page.create', {
    name: `${resourceName}`,
  })

  const onSuccess = () => {
    notify('ra.notification.created', 'info', { smart_count: 1 })
    redirect('list', basePath)
    refresh()
  }

  return (
    <Create
      title={<Title subTitle={title} />}
      {...props}
      onSuccess={onSuccess}
    >
      <SimpleForm
        redirect="list"
        variant={'outlined'}
        initialValues={{ public: true }}
        toolbar={<PlaylistCreateToolbar />}
      >
        <TextInput source="name" validate={required()} />
        <TextInput multiline source="comment" />
        {permissions === 'admin' && (
          <ReferenceInput
            source="ownerId"
            reference="user"
            perPage={0}
            sort={{ field: 'name', order: 'ASC' }}
          >
            <SelectInput
              label={'resources.playlist.fields.ownerName'}
              optionText="userName"
              optionValue="id"
              allowEmpty
            />
          </ReferenceInput>
        )}
        <BooleanInput source="public" initialValue={true} />
        <SmartPlaylistFields />
      </SimpleForm>
    </Create>
  )
}

export default PlaylistCreate
