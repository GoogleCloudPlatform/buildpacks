app = proc do |env|
  [500, {'Content-Type' => 'text/plain'}, ['FAIL']]
end

run app
