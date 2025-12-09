import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react'
import { AutocompleteArrayInput, useDataProvider, useInput } from 'react-admin'

const sanitizeChoices = (records = []) =>
  records
    .map((record) => ({
      id: record.id ?? record.name,
      name: record.name ?? record.id ?? '',
    }))
    .filter((record) => !!record.name)

const normalizeValues = (value) => {
  if (!value) return []
  return Array.isArray(value) ? value : [value]
}

const SmartCriteriaAutocompleteArrayInput = ({ reference, source, label }) => {
  const dataProvider = useDataProvider()
  const { input } = useInput({ source })
  const [choices, setChoices] = useState([])
  const inFlightRequest = useRef()

  const selectedChoices = useMemo(
    () => normalizeValues(input?.value).map((value) => ({ id: value, name: value })),
    [input?.value]
  )

  const mergedChoices = useMemo(() => {
    const choiceMap = new Map()
    sanitizeChoices(choices).forEach((choice) => choiceMap.set(choice.id, choice))
    selectedChoices.forEach((choice) => {
      if (!choiceMap.has(choice.id)) {
        choiceMap.set(choice.id, choice)
      }
    })
    return Array.from(choiceMap.values())
  }, [choices, selectedChoices])

  const fetchChoices = useCallback(
    async (searchText = '') => {
      if (inFlightRequest.current) {
        inFlightRequest.current.abort()
      }
      const controller = new AbortController()
      inFlightRequest.current = controller
      try {
        const { data } = await dataProvider.getList(reference, {
          filter: { name: [searchText] },
          pagination: { page: 1, perPage: 25 },
          sort: { field: 'name', order: 'ASC' },
          signal: controller.signal,
        })
        setChoices(sanitizeChoices(data))
      } catch (error) {
        if (error.name !== 'AbortError') {
          setChoices([])
        }
      }
    },
    [dataProvider, reference]
  )

  useEffect(() => {
    fetchChoices()
    return () => {
      if (inFlightRequest.current) {
        inFlightRequest.current.abort()
      }
    }
  }, [fetchChoices])

  return (
    <AutocompleteArrayInput
      source={source}
      label={label}
      optionText="name"
      optionValue="name"
      choices={mergedChoices}
      setFilter={fetchChoices}
      fullWidth
      helperText=" "
    />
  )
}

export default SmartCriteriaAutocompleteArrayInput
