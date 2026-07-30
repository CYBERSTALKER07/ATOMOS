require 'xcodeproj'
require 'find'

project_path = 'WarehouseAppIOS.xcodeproj'
project = Xcodeproj::Project.open(project_path)
target = project.targets.find { |t| t.name == 'WarehouseAppIOS' }
main_group = project.main_group.groups.find { |g| g.path == 'WarehouseApp' }

raise "Target not found" unless target
raise "Main group not found" unless main_group

source_folder = 'WarehouseApp'

added_count = 0

Find.find(source_folder) do |path|
  next unless File.file?(path) && path.end_with?('.swift')
  
  # Check if file is already in the project
  filename = File.basename(path)
  already_in_project = target.source_build_phase.files.any? do |build_file|
    build_file.file_ref && build_file.file_ref.path && build_file.file_ref.path.end_with?(filename)
  end

  unless already_in_project
    puts "Adding #{path} to project..."
    
    # We need to recreate the group hierarchy
    relative_path = path.sub("#{source_folder}/", '')
    path_components = relative_path.split('/')
    file_name = path_components.pop
    
    current_group = main_group
    path_components.each do |component|
      next_group = current_group.groups.find { |g| g.path == component || g.name == component }
      unless next_group
        next_group = current_group.new_group(component, component)
      end
      current_group = next_group
    end
    
    # Check if file_ref already exists in the group
    file_ref = current_group.files.find { |f| f.path == file_name }
    unless file_ref
      file_ref = current_group.new_file(file_name)
    end
    
    # Add to target
    target.add_file_references([file_ref])
    added_count += 1
  end
end

if added_count > 0
  project.save
  puts "Saved project with #{added_count} new files."
else
  puts "No new files to add."
end
