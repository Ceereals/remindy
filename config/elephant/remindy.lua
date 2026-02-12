--
-- Dynamic Remindy Menu for Elephant/Walker
--
Name = "remindy"
NamePretty = "Reminders"
Icon = "alarm-symbolic"
FixedOrder = true
RefreshOnChange = { os.getenv("HOME") .. "/.local/share/remindy/reminders.json" }

local reminders_file = os.getenv("HOME") .. "/.local/share/remindy/reminders.json"

local day_names = {
  [1] = "Mon", [2] = "Tue", [3] = "Wed", [4] = "Thu",
  [5] = "Fri", [6] = "Sat", [7] = "Sun",
}

function GetEntries()
  local entries = {}

  -- Fixed "Add reminder" entry at top
  table.insert(entries, {
    Text = "Add reminder",
    Subtext = "Create a new reminder",
    Icon = "list-add-symbolic",
    Terminal = true,
    Actions = {
      add = "remindy-add",
    },
  })

  -- Check file exists
  local f = io.open(reminders_file, "r")
  if not f then
    return entries
  end
  f:close()

  -- Read reminders via jq (tab-separated: id, type, time, days, text)
  local cmd = "jq -r '.reminders[] | [.id, .type, .time, (.days // [] | map(tostring) | join(\",\")), .text] | @tsv' '"
    .. reminders_file .. "' 2>/dev/null"
  local handle = io.popen(cmd)
  if not handle then
    return entries
  end

  for line in handle:lines() do
    local id, rtype, time, days, text = line:match("^([^\t]*)\t([^\t]*)\t([^\t]*)\t([^\t]*)\t(.+)$")
    if id and text then
      local schedule = ""

      if rtype == "once" then
        local date_part = time:sub(1, 10)
        local time_part = time:sub(12, 16)
        schedule = date_part .. " at " .. time_part
      elseif rtype == "daily" then
        schedule = "Every day at " .. time
      elseif rtype == "weekly" then
        local day_str = ""
        for d in days:gmatch("[^,]+") do
          local n = tonumber(d)
          if n and day_names[n] then
            if day_str ~= "" then day_str = day_str .. ", " end
            day_str = day_str .. day_names[n]
          end
        end
        schedule = "Every " .. day_str .. " at " .. time
      end

      table.insert(entries, {
        Text = text,
        Subtext = rtype .. " · " .. schedule,
        Icon = "alarm-symbolic",
        Actions = {
          delete = "remindy-remove " .. id,
        },
      })
    end
  end

  handle:close()
  return entries
end
