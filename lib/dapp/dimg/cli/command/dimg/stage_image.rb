module Dapp::Dimg::CLI
  module Command
    class Dimg < ::Dapp::CLI
      class StageImage < Base
        banner <<BANNER.freeze
Usage:

  dapp dimg stage image [options] [DIMG]

    DIMG                        Dapp image to process [default: *].

Options:
BANNER
        option :stage,
               long: '--stage STAGE',
               proc: proc { |v| v.to_sym },
               default: :docker_instructions,
               in: [:from, :before_install, :before_install_artifact, :g_a_archive, :g_a_pre_install_patch, :install,
                    :g_a_post_install_patch, :after_install_artifact, :before_setup, :before_setup_artifact, :g_a_pre_setup_patch,
                    :setup, :g_a_post_setup_patch, :after_setup_artifact, :g_a_latest_patch, :docker_instructions]
        def log_running_time
          false
        end

        def cli_options(**kwargs)
          super.tap do |config|
            config[:quiet] ||= !config[:verbose]
          end
        end
      end
    end
  end
end
