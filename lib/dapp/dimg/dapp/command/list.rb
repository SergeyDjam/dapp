module Dapp
  module Dimg
    module Dapp
      module Command
        module List
          def list
            build_configs.each do |config|
              if config._name.nil?
                log_warning('Dappfile has unnamed dimg!') if log_verbose?
              else
                puts config._name
              end
            end
          end
        end
      end
    end
  end # Dimg
end # Dapp
