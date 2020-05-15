# go-modiff fish shell completion

function __fish_go-modiff_no_subcommand --description 'Test if there has been any subcommand yet'
    for i in (commandline -opc)
        if contains -- $i docs d fish f help h
            return 1
        end
    end
    return 0
end

complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l repository -s r -r -d 'repository to be used, like: github.com/owner/repo'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l from -s f -r -d 'the start of the comparison, any valid git rev'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l to -s t -r -d 'the end of the comparison, any valid git rev'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l link -s l -d 'add diff links to the markdown output'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l header-level -s i -r -d 'add a higher markdown header level depth'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l debug -s d -d 'enable debug output'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l help -s h -d 'show help'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l version -s v -d 'print the version'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l help -s h -d 'show help'
complete -c go-modiff -n '__fish_go-modiff_no_subcommand' -f -l version -s v -d 'print the version'
complete -c go-modiff -n '__fish_seen_subcommand_from docs d' -f -l help -s h -d 'show help'
complete -r -c go-modiff -n '__fish_go-modiff_no_subcommand' -a 'docs d' -d 'generate the markdown or man page documentation and print it to stdout'
complete -c go-modiff -n '__fish_seen_subcommand_from docs d' -f -l markdown -d 'print the markdown version'
complete -c go-modiff -n '__fish_seen_subcommand_from docs d' -f -l man -d 'print the man version'
complete -c go-modiff -n '__fish_seen_subcommand_from fish f' -f -l help -s h -d 'show help'
complete -r -c go-modiff -n '__fish_go-modiff_no_subcommand' -a 'fish f' -d 'generate the fish shell completion'
complete -c go-modiff -n '__fish_seen_subcommand_from fish f' -f -l help -s h -d 'show help'
complete -c go-modiff -n '__fish_seen_subcommand_from help h' -f -l help -s h -d 'show help'
complete -r -c go-modiff -n '__fish_go-modiff_no_subcommand' -a 'help h' -d 'Shows a list of commands or help for one command'
