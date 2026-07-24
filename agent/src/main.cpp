#include "vps_pilot/agent.hpp"
#include "vps_pilot/config.hpp"

#include <filesystem>
#include <iostream>
#include <string>

namespace {

void print_usage(const char* program) {
    std::cout << "Usage: " << program << " [--config PATH]\n";
}

}  // namespace

int main(int argc, char** argv) {
    std::filesystem::path config_path{"/etc/vps-pilot-agent/config.json"};
    for (int index = 1; index < argc; ++index) {
        const std::string argument = argv[index];
        if (argument == "--config" && index + 1 < argc) {
            config_path = argv[++index];
        } else if (argument == "--help" || argument == "-h") {
            print_usage(argv[0]);
            return 0;
        } else {
            std::cerr << "Unknown argument: " << argument << '\n';
            print_usage(argv[0]);
            return 2;
        }
    }

    try {
        auto config = vps_pilot::Config::load(config_path);
        vps_pilot::Agent agent(std::move(config));
        agent.run();
    } catch (const std::exception& exception) {
        std::cerr << "fatal: " << exception.what() << '\n';
        return 1;
    }
}
