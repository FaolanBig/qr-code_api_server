{
  title: 'QR Code API',
  secure_tunnel: true,

  connection: {
    fields: [
      {
        name: 'base_url',
        label: 'API base URL',
        type: 'string',
        control_type: 'url',
        optional: false,
        hint: 'The service origin, for example https://qr.example.com'
      },
      {
        name: 'api_key',
        label: 'API key',
        type: 'string',
        control_type: 'password',
        optional: false,
        hint: 'The API key configured as API_KEY on the server. It must contain at least 16 characters.'
      }
    ],

    authorization: {
      type: 'custom_auth',
      apply: lambda do |connection|
        headers('X-API-Key' => connection['api_key'].to_s)
      end
    },

    base_uri: lambda do |connection|
      base_url = connection['base_url'].to_s.strip
      base_url = "https://#{base_url}" unless base_url.match?(%r{\Ahttps?://}i)
      base_url.sub(%r{/+\z}, '')
    end
  },

  test: lambda do |_connection|
    get('/healthz')
  end,

  object_definitions: {
    qr_code: {
      fields: lambda do |_connection = nil, _config_fields = nil|
        [
          { name: 'type', type: 'string', label: 'QR payload type' },
          { name: 'format', type: 'string', label: 'Image format' },
          { name: 'size', type: 'integer', label: 'PNG size' },
          { name: 'recovery', type: 'string', label: 'Error recovery level' },
          { name: 'logo_included', type: 'boolean', label: 'Logo included' },
          { name: 'logo_source', type: 'string', label: 'Logo source' },
          { name: 'png_base64', type: 'string', label: 'PNG (Base64)' },
          { name: 'data_uri', type: 'string', label: 'PNG data URI' }
        ]
      end
    }
  },

  actions: {
    check_health: {
      title: 'Check API health',
      description: 'Check whether the API process is healthy.',

      execute: lambda do |_connection, _input|
        get('/healthz')
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'status', type: 'string', label: 'Health status' }]
      end
    },

    check_readiness: {
      title: 'Check API readiness',
      description: 'Check whether the API process is ready to serve requests.',

      execute: lambda do |_connection, _input|
        get('/readyz')
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'status', type: 'string', label: 'Readiness status' }]
      end
    },

    get_openapi_specification: {
      title: 'Get OpenAPI specification',
      description: 'Retrieve the public OpenAPI 3.1 specification served by the API.',

      execute: lambda do |_connection, _input|
        get('/openapi.yaml')
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'openapi', type: 'string', label: 'OpenAPI specification' }]
      end
    },

    get_api_index: {
      title: 'Get API index',
      description: 'Retrieve the API name, version, and available QR-code endpoints.',

      execute: lambda do |_connection, _input|
        get('/api/v1/')
      end,

      output_fields: lambda do |_object_definitions = nil|
        [
          { name: 'name', type: 'string', label: 'API name' },
          { name: 'version', type: 'string', label: 'API version' },
          { name: 'endpoints', type: 'array', label: 'Endpoints' }
        ]
      end
    },

    create_url_qr: {
      title: 'Create URL QR code',
      description: 'Generate a PNG QR code for an absolute HTTP or HTTPS URL.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'url', label: 'URL', type: 'string', control_type: 'url', optional: false },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256, hint: '128 to 2048 pixels.' },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true, hint: 'Base64-encoded PNG. Used for JSON requests.' },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true, hint: 'Optional PNG upload. When supplied, the request is sent as multipart/form-data.' }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'url' => input['url'],
          'size' => input['size'],
          'recovery' => input['recovery'],
          'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }

        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/url', payload).request_format_multipart_form
        else
          post('/api/v1/qr/url', payload)
        end
      end,

      output_fields: lambda do |object_definitions = nil|
        if object_definitions.is_a?(Hash) && object_definitions['qr_code']
          [{ name: 'qr_code', type: 'object', properties: object_definitions['qr_code'] }]
        else
          [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
            { name: 'type', type: 'string' },
            { name: 'format', type: 'string' },
            { name: 'size', type: 'integer' },
            { name: 'recovery', type: 'string' },
            { name: 'logo_included', type: 'boolean' },
            { name: 'logo_source', type: 'string' },
            { name: 'png_base64', type: 'string' },
            { name: 'data_uri', type: 'string' }
          ] }]
        end
      end
    },

    create_vcard_qr: {
      title: 'Create vCard QR code',
      description: 'Generate a PNG QR code for a contact record.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'first_name', label: 'First name', type: 'string', optional: true, hint: 'At least one of first name or last name is required.' },
          { name: 'last_name', label: 'Last name', type: 'string', optional: true },
          { name: 'organization', label: 'Organization', type: 'string', optional: true },
          { name: 'title', label: 'Job title', type: 'string', optional: true },
          { name: 'phone', label: 'Phone', type: 'string', optional: true },
          { name: 'email', label: 'Email', type: 'string', optional: true },
          { name: 'website', label: 'Website', type: 'string', control_type: 'url', optional: true },
          { name: 'address', label: 'Address', type: 'string', optional: true },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'first_name' => input['first_name'], 'last_name' => input['last_name'],
          'organization' => input['organization'], 'title' => input['title'],
          'phone' => input['phone'], 'email' => input['email'],
          'website' => input['website'], 'address' => input['address'],
          'size' => input['size'], 'recovery' => input['recovery'],
          'show_logo' => input['show_logo'], 'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/vcard', payload).request_format_multipart_form
        else
          post('/api/v1/qr/vcard', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_wifi_qr: {
      title: 'Create Wi-Fi QR code',
      description: 'Generate a PNG QR code containing Wi-Fi network configuration.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'ssid', label: 'Network name (SSID)', type: 'string', optional: false },
          { name: 'password', label: 'Password', type: 'string', control_type: 'password', optional: true },
          { name: 'security', label: 'Security', type: 'string', control_type: 'select', optional: true, default: 'WPA', pick_list: [%w[WPA WPA], %w[WEP WEP], %w[NOPASS NOPASS]] },
          { name: 'hidden', label: 'Hidden network', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'ssid' => input['ssid'], 'password' => input['password'],
          'security' => input['security'], 'hidden' => input['hidden'],
          'size' => input['size'], 'recovery' => input['recovery'],
          'show_logo' => input['show_logo'], 'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/wifi', payload).request_format_multipart_form
        else
          post('/api/v1/qr/wifi', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_calendar_qr: {
      title: 'Create calendar event QR code',
      description: 'Generate a PNG QR code for an iCalendar event.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'title', label: 'Event title', type: 'string', optional: false },
          { name: 'description', label: 'Description', type: 'string', optional: true },
          { name: 'location', label: 'Location', type: 'string', optional: true },
          { name: 'start', label: 'Start', type: 'string', optional: false, hint: 'RFC 3339 timestamp or YYYY-MM-DD for an all-day event.' },
          { name: 'end', label: 'End', type: 'string', optional: true, hint: 'RFC 3339 timestamp or YYYY-MM-DD for an all-day event.' },
          { name: 'timezone', label: 'Timezone', type: 'string', optional: true, hint: 'For example Europe/London.' },
          { name: 'all_day', label: 'All-day event', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'title' => input['title'], 'description' => input['description'],
          'location' => input['location'], 'start' => input['start'],
          'end' => input['end'], 'timezone' => input['timezone'],
          'all_day' => input['all_day'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/calendar', payload).request_format_multipart_form
        else
          post('/api/v1/qr/calendar', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_text_qr: {
      title: 'Create text QR code',
      description: 'Generate a PNG QR code containing plain text.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'text', label: 'Text', type: 'string', control_type: 'text', optional: false },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'text' => input['text'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/text', payload).request_format_multipart_form
        else
          post('/api/v1/qr/text', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_file_qr: {
      title: 'Create file data URI QR code',
      description: 'Generate a PNG QR code for a small file encoded as a data URI.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'filename', label: 'Filename', type: 'string', optional: false },
          { name: 'mime_type', label: 'MIME type', type: 'string', optional: true, default: 'application/octet-stream' },
          { name: 'base64_content', label: 'File content (Base64)', type: 'string', control_type: 'text', optional: false, hint: 'Decoded content limit is 1024 bytes. Base64 input limit is 4096 characters.' },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'filename' => input['filename'], 'mime_type' => input['mime_type'],
          'base64_content' => input['base64_content'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/file', payload).request_format_multipart_form
        else
          post('/api/v1/qr/file', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_number_qr: {
      title: 'Create number QR code',
      description: 'Generate a PNG QR code containing a number or telephone-style value.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'number', label: 'Number', type: 'string', optional: false, hint: 'Digits and telephone-style formatting are accepted.' },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'number' => input['number'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/number', payload).request_format_multipart_form
        else
          post('/api/v1/qr/number', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_email_qr: {
      title: 'Create email QR code',
      description: 'Generate a PNG QR code that opens a compose-email action.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'email', label: 'Email address', type: 'string', optional: false },
          { name: 'subject', label: 'Subject', type: 'string', optional: true },
          { name: 'body', label: 'Email body', type: 'string', control_type: 'text', optional: true },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'email' => input['email'], 'subject' => input['subject'],
          'body' => input['body'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/email', payload).request_format_multipart_form
        else
          post('/api/v1/qr/email', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_sms_qr: {
      title: 'Create SMS QR code',
      description: 'Generate a PNG QR code that opens an SMS compose action.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'phone', label: 'Phone number', type: 'string', optional: false },
          { name: 'body', label: 'Message', type: 'string', control_type: 'text', optional: true },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'phone' => input['phone'], 'body' => input['body'],
          'size' => input['size'], 'recovery' => input['recovery'],
          'show_logo' => input['show_logo'], 'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/sms', payload).request_format_multipart_form
        else
          post('/api/v1/qr/sms', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    },

    create_geolocation_qr: {
      title: 'Create geolocation QR code',
      description: 'Generate a PNG QR code containing geographic coordinates.',

      input_fields: lambda do |_object_definitions = nil|
        [
          { name: 'latitude', label: 'Latitude', type: 'number', control_type: 'number', optional: false, hint: '-90 to 90.' },
          { name: 'longitude', label: 'Longitude', type: 'number', control_type: 'number', optional: false, hint: '-180 to 180.' },
          { name: 'label', label: 'Location label', type: 'string', optional: true },
          { name: 'size', label: 'PNG size', type: 'integer', control_type: 'number', optional: true, default: 256 },
          { name: 'recovery', label: 'Error recovery', type: 'string', control_type: 'select', optional: true, default: 'medium', pick_list: [%w[Low low], %w[Medium medium], %w[High high], %w[Highest highest]] },
          { name: 'show_logo', label: 'Show logo', type: 'boolean', control_type: 'checkbox', optional: true, default: false },
          { name: 'logo_png_base64', label: 'Custom logo (Base64)', type: 'string', control_type: 'text', optional: true },
          { name: 'logo', label: 'Custom logo file', type: 'string', control_type: 'file', optional: true }
        ]
      end,

      execute: lambda do |_connection, input|
        payload = {
          'latitude' => input['latitude'], 'longitude' => input['longitude'],
          'label' => input['label'], 'size' => input['size'],
          'recovery' => input['recovery'], 'show_logo' => input['show_logo'],
          'logo_png_base64' => input['logo_png_base64']
        }
        payload.delete_if { |_key, value| value.nil? || (value.is_a?(String) && value.empty?) }
        logo = input['logo']
        unless logo.nil? || (logo.is_a?(String) && logo.empty?)
          payload.delete('logo_png_base64')
          payload['logo'] = logo
          post('/api/v1/qr/geolocation', payload).request_format_multipart_form
        else
          post('/api/v1/qr/geolocation', payload)
        end
      end,

      output_fields: lambda do |_object_definitions = nil|
        [{ name: 'qr_code', type: 'object', label: 'Generated QR code', properties: [
          { name: 'type', type: 'string' }, { name: 'format', type: 'string' },
          { name: 'size', type: 'integer' }, { name: 'recovery', type: 'string' },
          { name: 'logo_included', type: 'boolean' }, { name: 'logo_source', type: 'string' },
          { name: 'png_base64', type: 'string' }, { name: 'data_uri', type: 'string' }
        ] }]
      end
    }
  }
}
