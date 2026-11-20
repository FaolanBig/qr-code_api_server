require 'minitest/autorun'

class ConnectorTest < Minitest::Test
  CONNECTOR = eval(File.read(File.join(__dir__, 'connector.rb')), binding,
                   File.join(__dir__, 'connector.rb'))

  class Request
    def request_format_multipart_form
      self
    end
  end

  class Context
    attr_reader :path, :payload

    def post(path, payload)
      @path = path
      @payload = payload
      Request.new
    end
  end

  def test_url_qr_converts_workato_string_values
    context = Context.new
    action = CONNECTOR[:actions][:create_url_qr]
    context.instance_exec({}, {
      'url' => 'https://example.com', 'size' => '256',
      'recovery' => 'medium', 'show_logo' => 'false'
    }, &action[:execute])

    assert_equal '/api/v1/qr/url', context.path
    assert_equal 256, context.payload['size']
    assert_instance_of Integer, context.payload['size']
    assert_equal false, context.payload['show_logo']
  end

  def test_boolean_and_float_values_are_normalized
    context = Context.new
    context.instance_exec({}, {
      'ssid' => 'Office', 'hidden' => 'true', 'size' => '512',
      'show_logo' => 'true'
    }, &CONNECTOR[:actions][:create_wifi_qr][:execute])

    assert_equal true, context.payload['hidden']
    assert_equal 512, context.payload['size']
    assert_equal true, context.payload['show_logo']

    context.instance_exec({}, {
      'latitude' => '51.5074', 'longitude' => '-0.1278',
      'size' => '256', 'show_logo' => 'false'
    }, &CONNECTOR[:actions][:create_geolocation_qr][:execute])

    assert_in_delta 51.5074, context.payload['latitude']
    assert_in_delta(-0.1278, context.payload['longitude'])
    assert_equal false, context.payload['show_logo']
  end
end
