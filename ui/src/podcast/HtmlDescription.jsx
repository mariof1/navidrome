import React, { useMemo } from 'react'
import DOMPurify from 'dompurify'
import { Typography } from '@material-ui/core'

const allowedTags = ['p', 'br', 'ul', 'ol', 'li', 'em', 'strong', 'a']

const sanitize = (value) => {
  if (!value) return ''
  const normalized = value.replace(/\n/g, '<br />')
  return DOMPurify.sanitize(normalized, {
    ALLOWED_TAGS: allowedTags,
    ALLOWED_ATTR: ['href'],
  })
}

const HtmlDescription = ({ value, variant = 'body2', className }) => {
  const safeHtml = useMemo(() => sanitize(value), [value])

  if (!safeHtml.trim()) {
    return null
  }

  return (
    <Typography
      component="div"
      variant={variant}
      className={className}
      style={{ wordBreak: 'break-word' }}
    >
      <span dangerouslySetInnerHTML={{ __html: safeHtml }} />
    </Typography>
  )
}

export default HtmlDescription
